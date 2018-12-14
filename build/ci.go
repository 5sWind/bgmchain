//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

//

/*
The ci command is called from Continuous Integration scripts.

Usage: go run ci.go <command> <command flags/arguments>

Available commands are:

   install    [ -arch architecture ] [ packages... ]                                           -- builds packages and executables
   test       [ -coverage ] [ packages... ]                                                    -- runs the tests
   lint                                                                                        -- runs certain pre-selected linters
   archive    [ -arch architecture ] [ -type zip|tar ] [ -signer key-envvar ] [ -upload dest ] -- archives build artefacts
   importkeys                                                                                  -- imports signing keys from env
   debsrc     [ -signer key-id ] [ -upload dest ]                                              -- creates a debian source package
   nsis                                                                                        -- creates a Windows NSIS installer
   aar        [ -local ] [ -sign key-id ] [-deploy repo] [ -upload dest ]                      -- creates an Android archive
   xcode      [ -local ] [ -sign key-id ] [-deploy repo] [ -upload dest ]                      -- creates an iOS XCode framework
   xgo        [ -alltools ] [ options ]                                                        -- cross builds according to options
   purge      [ -store blobstore ] [ -days threshold ]                                         -- purges old archives from the blobstore

For all commands, -n prevents execution of external programs (dry run mode).

*/
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/5sWind/bgmchain/internal/build"
)

var (
//
	gbgmArchiveFiles = []string{
		"COPYING",
		executablePath("gbgm"),
	}

//
	allToolsArchiveFiles = []string{
		"COPYING",
		executablePath("abigen"),
		executablePath("bootnode"),
		executablePath("evm"),
		executablePath("gbgm"),
		executablePath("puppbgm"),
		executablePath("rlpdump"),
		executablePath("swarm"),
		executablePath("wnode"),
	}

//
	debExecutables = []debExecutable{
		{
			Name:        "abigen",
			Description: "Source code generator to convert Bgmchain contract definitions into easy to use, compile-time type-safe Go packages.",
		},
		{
			Name:        "bootnode",
			Description: "Bgmchain bootnode.",
		},
		{
			Name:        "evm",
			Description: "Developer utility version of the EVM (Bgmchain Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode.",
		},
		{
			Name:        "gbgm",
			Description: "Bgmchain CLI client.",
		},
		{
			Name:        "puppbgm",
			Description: "Bgmchain private network manager.",
		},
		{
			Name:        "rlpdump",
			Description: "Developer utility tool that prints RLP structures.",
		},
		{
			Name:        "swarm",
			Description: "Bgmchain Swarm daemon and tools",
		},
		{
			Name:        "wnode",
			Description: "Bgmchain Whisper diagnostic tool",
		},
	}

//.
//
//
//
	debDistros = []string{"trusty", "xenial", "zesty", "artful"}
)

var GOBIN, _ = filepath.Abs(filepath.Join("build", "bin"))

func executablePath(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(GOBIN, name)
}

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("build", "ci.go")); os.IsNotExist(err) {
		log.Fatal("this script must be run from the root of the repository")
	}
	if len(os.Args) < 2 {
		log.Fatal("need subcommand as first argument")
	}
	switch os.Args[1] {
	case "install":
		doInstall(os.Args[2:])
	case "test":
		doTest(os.Args[2:])
	case "lint":
		doLint(os.Args[2:])
	case "archive":
		doArchive(os.Args[2:])
	case "debsrc":
		doDebianSource(os.Args[2:])
	case "nsis":
		doWindowsInstaller(os.Args[2:])
	case "aar":
		doAndroidArchive(os.Args[2:])
	case "xcode":
		doXCodeFramework(os.Args[2:])
	case "xgo":
		doXgo(os.Args[2:])
	case "purge":
		doPurge(os.Args[2:])
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

//

func doInstall(cmdline []string) {
	var (
		arch = flag.String("arch", "", "Architecture to cross build for")
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

//
//
	if !strings.Contains(runtime.Version(), "devel") {
		var minor int
		fmt.Sscanf(strings.TrimPrefix(runtime.Version(), "go1."), "%d", &minor)
		if minor < 7 {
			log.Println("You have Go version", runtime.Version())
			log.Println("bgmchain requires at least Go version 1.7 and cannot")
			log.Println("be compiled with an earlier version. Please upgrade your Go installation.")
			os.Exit(1)
		}
	}
//
	packages := []string{"./..."}
	if flag.NArg() > 0 {
		packages = flag.Args()
	}
	packages = build.ExpandPackagesNoVendor(packages)

	if *arch == "" || *arch == runtime.GOARCH {
		goinstall := goTool("install", buildFlags(env)...)
		goinstall.Args = append(goinstall.Args, "-v")
		goinstall.Args = append(goinstall.Args, packages...)
		build.MustRun(goinstall)
		return
	}
//
	if *arch == "arm" {
		os.RemoveAll(filepath.Join(runtime.GOROOT(), "pkg", runtime.GOOS+"_arm"))
		for _, path := range filepath.SplitList(build.GOPATH()) {
			os.RemoveAll(filepath.Join(path, "pkg", runtime.GOOS+"_arm"))
		}
	}
//
	goinstall := goToolArch(*arch, "install", buildFlags(env)...)
	goinstall.Args = append(goinstall.Args, "-v")
	goinstall.Args = append(goinstall.Args, []string{"-buildmode", "archive"}...)
	goinstall.Args = append(goinstall.Args, packages...)
	build.MustRun(goinstall)

	if cmds, err := ioutil.ReadDir("cmd"); err == nil {
		for _, cmd := range cmds {
			pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Join(".", "cmd", cmd.Name()), nil, parser.PackageClauseOnly)
			if err != nil {
				log.Fatal(err)
			}
			for name := range pkgs {
				if name == "main" {
					gobuild := goToolArch(*arch, "build", buildFlags(env)...)
					gobuild.Args = append(gobuild.Args, "-v")
					gobuild.Args = append(gobuild.Args, []string{"-o", executablePath(cmd.Name())}...)
					gobuild.Args = append(gobuild.Args, "."+string(filepath.Separator)+filepath.Join("cmd", cmd.Name()))
					build.MustRun(gobuild)
					break
				}
			}
		}
	}
}

func buildFlags(env build.Environment) (flags []string) {
	var ld []string
	if env.Commit != "" {
		ld = append(ld, "-X", "main.gitCommit="+env.Commit)
	}
	if runtime.GOOS == "darwin" {
		ld = append(ld, "-s")
	}

	if len(ld) > 0 {
		flags = append(flags, "-ldflags", strings.Join(ld, " "))
	}
	return flags
}

func goTool(subcmd string, args ...string) *exec.Cmd {
	return goToolArch(runtime.GOARCH, subcmd, args...)
}

func goToolArch(arch string, subcmd string, args ...string) *exec.Cmd {
	cmd := build.GoTool(subcmd, args...)
	if subcmd == "build" || subcmd == "install" || subcmd == "test" {
//
//
		if runtime.GOOS == "windows" && runtime.Version() < "go1.8" {
			cmd.Args = append(cmd.Args, []string{"-ldflags", "-extldflags -Wl,--allow-multiple-definition"}...)
		}
	}
	cmd.Env = []string{"GOPATH=" + build.GOPATH()}
	if arch == "" || arch == runtime.GOARCH {
		cmd.Env = append(cmd.Env, "GOBIN="+GOBIN)
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
		cmd.Env = append(cmd.Env, "GOARCH="+arch)
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") || strings.HasPrefix(e, "GOBIN=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}

//
//
//

func doTest(cmdline []string) {
	var (
		coverage = flag.Bool("coverage", false, "Whbgmchain to record code coverage")
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	packages := []string{"./..."}
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	}
	packages = build.ExpandPackagesNoVendor(packages)

//
	build.MustRun(goTool("vet", packages...))

//
	gotest := goTool("test", buildFlags(env)...)
//
//
	gotest.Args = append(gotest.Args, "-p", "1")
	if *coverage {
		gotest.Args = append(gotest.Args, "-covermode=atomic", "-cover")
	}

	gotest.Args = append(gotest.Args, packages...)
	build.MustRun(gotest)
}

//
func doLint(cmdline []string) {
	flag.CommandLine.Parse(cmdline)

	packages := []string{"./..."}
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	}
//
	build.MustRun(goTool("get", "gopkg.in/alecthomas/gometalinter.v1"))
	build.MustRunCommand(filepath.Join(GOBIN, "gometalinter.v1"), "--install")

//
	configs := []string{"--vendor", "--disable-all", "--enable=vet", "--enable=gofmt", "--enable=misspell"}
	build.MustRunCommand(filepath.Join(GOBIN, "gometalinter.v1"), append(configs, packages...)...)

//
	for _, linter := range []string{"unconvert"} {
		configs = []string{"--vendor", "--deadline=10m", "--disable-all", "--enable=" + linter}
		build.MustRunCommand(filepath.Join(GOBIN, "gometalinter.v1"), append(configs, packages...)...)
	}
}

//

func doArchive(cmdline []string) {
	var (
		arch   = flag.String("arch", runtime.GOARCH, "Architecture cross packaging")
		atype  = flag.String("type", "zip", "Type of archive to write (zip|tar)")
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. LINUX_SIGNING_KEY)`)
		upload = flag.String("upload", "", `Destination to upload the archives (usually "gbgmstore/builds")`)
		ext    string
	)
	flag.CommandLine.Parse(cmdline)
	switch *atype {
	case "zip":
		ext = ".zip"
	case "tar":
		ext = ".tar.gz"
	default:
		log.Fatal("unknown archive type: ", atype)
	}

	var (
		env      = build.Env()
		base     = archiveBasename(*arch, env)
		gbgm     = "gbgm-" + base + ext
		alltools = "gbgm-alltools-" + base + ext
	)
	maybeSkipArchive(env)
	if err := build.WriteArchive(gbgm, gbgmArchiveFiles); err != nil {
		log.Fatal(err)
	}
	if err := build.WriteArchive(alltools, allToolsArchiveFiles); err != nil {
		log.Fatal(err)
	}
	for _, archive := range []string{gbgm, alltools} {
		if err := archiveUpload(archive, *upload, *signer); err != nil {
			log.Fatal(err)
		}
	}
}

func archiveBasename(arch string, env build.Environment) string {
	platform := runtime.GOOS + "-" + arch
	if arch == "arm" {
		platform += os.Getenv("GOARM")
	}
	if arch == "android" {
		platform = "android-all"
	}
	if arch == "ios" {
		platform = "ios-all"
	}
	return platform + "-" + archiveVersion(env)
}

func archiveVersion(env build.Environment) string {
	version := build.VERSION()
	if isUnstableBuild(env) {
		version += "-unstable"
	}
	if env.Commit != "" {
		version += "-" + env.Commit[:8]
	}
	return version
}

func archiveUpload(archive string, blobstore string, signer string) error {
//
	if signer != "" {
		pgpkey, err := base64.StdEncoding.DecodeString(os.Getenv(signer))
		if err != nil {
			return fmt.Errorf("invalid base64 %s", signer)
		}
		if err := build.PGPSignFile(archive, archive+".asc", string(pgpkey)); err != nil {
			return err
		}
	}
//
	if blobstore != "" {
		auth := build.AzureBlobstoreConfig{
			Account:   strings.Split(blobstore, "/")[0],
			Token:     os.Getenv("AZURE_BLOBSTORE_TOKEN"),
			Container: strings.SplitN(blobstore, "/", 2)[1],
		}
		if err := build.AzureBlobstoreUpload(archive, filepath.Base(archive), auth); err != nil {
			return err
		}
		if signer != "" {
			if err := build.AzureBlobstoreUpload(archive+".asc", filepath.Base(archive+".asc"), auth); err != nil {
				return err
			}
		}
	}
	return nil
}

//
func maybeSkipArchive(env build.Environment) {
	if env.IsPullRequest {
		log.Printf("skipping because this is a PR build")
		os.Exit(0)
	}
	if env.IsCronJob {
		log.Printf("skipping because this is a cron job")
		os.Exit(0)
	}
	if env.Branch != "master" && !strings.HasPrefix(env.Tag, "v1.") {
		log.Printf("skipping because branch %q, tag %q is not on the whitelist", env.Branch, env.Tag)
		os.Exit(0)
	}
}

//

func doDebianSource(cmdline []string) {
	var (
		signer  = flag.String("signer", "", `Signing key name, also used as package author`)
		upload  = flag.String("upload", "", `Where to upload the source package (usually "ppa:bgmchain/bgmchain")`)
		workdir = flag.String("workdir", "", `Output directory for packages (uses temp dir if unset)`)
		now     = time.Now()
	)
	flag.CommandLine.Parse(cmdline)
	*workdir = makeWorkdir(*workdir)
	env := build.Env()
	maybeSkipArchive(env)

//
	if b64key := os.Getenv("PPA_SIGNING_KEY"); b64key != "" {
		key, err := base64.StdEncoding.DecodeString(b64key)
		if err != nil {
			log.Fatal("invalid base64 PPA_SIGNING_KEY")
		}
		gpg := exec.Command("gpg", "--import")
		gpg.Stdin = bytes.NewReader(key)
		build.MustRun(gpg)
	}

//
	for _, distro := range debDistros {
		meta := newDebMetadata(distro, *signer, env, now)
		pkgdir := stageDebianSource(*workdir, meta)
		debuild := exec.Command("debuild", "-S", "-sa", "-us", "-uc")
		debuild.Dir = pkgdir
		build.MustRun(debuild)

		changes := fmt.Sprintf("%s_%s_source.changes", meta.Name(), meta.VersionString())
		changes = filepath.Join(*workdir, changes)
		if *signer != "" {
			build.MustRunCommand("debsign", changes)
		}
		if *upload != "" {
			build.MustRunCommand("dput", *upload, changes)
		}
	}
}

func makeWorkdir(wdflag string) string {
	var err error
	if wdflag != "" {
		err = os.MkdirAll(wdflag, 0744)
	} else {
		wdflag, err = ioutil.TempDir("", "gbgm-build-")
	}
	if err != nil {
		log.Fatal(err)
	}
	return wdflag
}

func isUnstableBuild(env build.Environment) bool {
	if env.Tag != "" {
		return false
	}
	return true
}

type debMetadata struct {
	Env build.Environment

//
//
//
	Version string

	Author       string //
	Distro, Time string
	Executables  []debExecutable
}

type debExecutable struct {
	Name, Description string
}

func newDebMetadata(distro, author string, env build.Environment, t time.Time) debMetadata {
	if author == "" {
//
		author = "Bgmchain Builds <fjl@bgmchain.org>"
	}
	return debMetadata{
		Env:         env,
		Author:      author,
		Distro:      distro,
		Version:     build.VERSION(),
		Time:        t.Format(time.RFC1123Z),
		Executables: debExecutables,
	}
}

//
//
func (meta debMetadata) Name() string {
	if isUnstableBuild(meta.Env) {
		return "bgmchain-unstable"
	}
	return "bgmchain"
}

//
func (meta debMetadata) VersionString() string {
	vsn := meta.Version
	if meta.Env.Buildnum != "" {
		vsn += "+build" + meta.Env.Buildnum
	}
	if meta.Distro != "" {
		vsn += "+" + meta.Distro
	}
	return vsn
}

//
func (meta debMetadata) ExeList() string {
	names := make([]string, len(meta.Executables))
	for i, e := range meta.Executables {
		names[i] = meta.ExeName(e)
	}
	return strings.Join(names, ", ")
}

//
func (meta debMetadata) ExeName(exe debExecutable) string {
	if isUnstableBuild(meta.Env) {
		return exe.Name + "-unstable"
	}
	return exe.Name
}

//
//
func (meta debMetadata) ExeConflicts(exe debExecutable) string {
	if isUnstableBuild(meta.Env) {
//
//
		//
//
//
//
//
//
		return "bgmchain, " + exe.Name
	}
	return ""
}

func stageDebianSource(tmpdir string, meta debMetadata) (pkgdir string) {
	pkg := meta.Name() + "-" + meta.VersionString()
	pkgdir = filepath.Join(tmpdir, pkg)
	if err := os.Mkdir(pkgdir, 0755); err != nil {
		log.Fatal(err)
	}

//
	build.MustRunCommand("git", "checkout-index", "-a", "--prefix", pkgdir+string(filepath.Separator))

//
	debian := filepath.Join(pkgdir, "debian")
	build.Render("build/deb.rules", filepath.Join(debian, "rules"), 0755, meta)
	build.Render("build/deb.changelog", filepath.Join(debian, "changelog"), 0644, meta)
	build.Render("build/deb.control", filepath.Join(debian, "control"), 0644, meta)
	build.Render("build/deb.copyright", filepath.Join(debian, "copyright"), 0644, meta)
	build.RenderString("8\n", filepath.Join(debian, "compat"), 0644, meta)
	build.RenderString("3.0 (native)\n", filepath.Join(debian, "source/format"), 0644, meta)
	for _, exe := range meta.Executables {
		install := filepath.Join(debian, meta.ExeName(exe)+".install")
		docs := filepath.Join(debian, meta.ExeName(exe)+".docs")
		build.Render("build/deb.install", install, 0644, exe)
		build.Render("build/deb.docs", docs, 0644, exe)
	}

	return pkgdir
}

//

func doWindowsInstaller(cmdline []string) {
//
	var (
		arch    = flag.String("arch", runtime.GOARCH, "Architecture for cross build packaging")
		signer  = flag.String("signer", "", `Environment variable holding the signing key (e.g. WINDOWS_SIGNING_KEY)`)
		upload  = flag.String("upload", "", `Destination to upload the archives (usually "gbgmstore/builds")`)
		workdir = flag.String("workdir", "", `Output directory for packages (uses temp dir if unset)`)
	)
	flag.CommandLine.Parse(cmdline)
	*workdir = makeWorkdir(*workdir)
	env := build.Env()
	maybeSkipArchive(env)

//
	var (
		devTools []string
		allTools []string
		gbgmTool string
	)
	for _, file := range allToolsArchiveFiles {
		if file == "COPYING" { //
			continue
		}
		allTools = append(allTools, filepath.Base(file))
		if filepath.Base(file) == "gbgm.exe" {
			gbgmTool = file
		} else {
			devTools = append(devTools, file)
		}
	}

//
//
	templateData := map[string]interface{}{
		"License":  "COPYING",
		"Gbgm":     gbgmTool,
		"DevTools": devTools,
	}
	build.Render("build/nsis.gbgm.nsi", filepath.Join(*workdir, "gbgm.nsi"), 0644, nil)
	build.Render("build/nsis.install.nsh", filepath.Join(*workdir, "install.nsh"), 0644, templateData)
	build.Render("build/nsis.uninstall.nsh", filepath.Join(*workdir, "uninstall.nsh"), 0644, allTools)
	build.Render("build/nsis.pathupdate.nsh", filepath.Join(*workdir, "PathUpdate.nsh"), 0644, nil)
	build.Render("build/nsis.envvarupdate.nsh", filepath.Join(*workdir, "EnvVarUpdate.nsh"), 0644, nil)
	build.CopyFile(filepath.Join(*workdir, "SimpleFC.dll"), "build/nsis.simplefc.dll", 0755)
	build.CopyFile(filepath.Join(*workdir, "COPYING"), "COPYING", 0755)

//
//
//
	version := strings.Split(build.VERSION(), ".")
	if env.Commit != "" {
		version[2] += "-" + env.Commit[:8]
	}
	installer, _ := filepath.Abs("gbgm-" + archiveBasename(*arch, env) + ".exe")
	build.MustRunCommand("makensis.exe",
		"/DOUTPUTFILE="+installer,
		"/DMAJORVERSION="+version[0],
		"/DMINORVERSION="+version[1],
		"/DBUILDVERSION="+version[2],
		"/DARCH="+*arch,
		filepath.Join(*workdir, "gbgm.nsi"),
	)

//
	if err := archiveUpload(installer, *upload, *signer); err != nil {
		log.Fatal(err)
	}
}

//

func doAndroidArchive(cmdline []string) {
	var (
		local  = flag.Bool("local", false, `Flag whbgmchain we're only doing a local build (skip Maven artifacts)`)
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. ANDROID_SIGNING_KEY)`)
		deploy = flag.String("deploy", "", `Destination to deploy the archive (usually "https://oss.sonatype.org")`)
		upload = flag.String("upload", "", `Destination to upload the archive (usually "gbgmstore/builds")`)
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

//
	if os.Getenv("ANDROID_HOME") == "" {
		log.Fatal("Please ensure ANDROID_HOME points to your Android SDK")
	}
	if os.Getenv("ANDROID_NDK") == "" {
		log.Fatal("Please ensure ANDROID_NDK points to your Android NDK")
	}
//
	build.MustRun(goTool("get", "golang.org/x/mobile/cmd/gomobile"))
	build.MustRun(gomobileTool("init", "--ndk", os.Getenv("ANDROID_NDK")))
	build.MustRun(gomobileTool("bind", "--target", "android", "--javapkg", "org.bgmchain", "-v", "github.com/5sWind/bgmchain/mobile"))

	if *local {
//
		os.Rename("gbgm.aar", filepath.Join(GOBIN, "gbgm.aar"))
		return
	}
	meta := newMavenMetadata(env)
	build.Render("build/mvn.pom", meta.Package+".pom", 0755, meta)

//
	maybeSkipArchive(env)

//
	archive := "gbgm-" + archiveBasename("android", env) + ".aar"
	os.Rename("gbgm.aar", archive)

	if err := archiveUpload(archive, *upload, *signer); err != nil {
		log.Fatal(err)
	}
//
	os.Rename(archive, meta.Package+".aar")
	if *signer != "" && *deploy != "" {
//
		if b64key := os.Getenv(*signer); b64key != "" {
			key, err := base64.StdEncoding.DecodeString(b64key)
			if err != nil {
				log.Fatalf("invalid base64 %s", *signer)
			}
			gpg := exec.Command("gpg", "--import")
			gpg.Stdin = bytes.NewReader(key)
			build.MustRun(gpg)
		}
//
		repo := *deploy + "/service/local/staging/deploy/maven2"
		if meta.Develop {
			repo = *deploy + "/content/repositories/snapshots"
		}
		build.MustRunCommand("mvn", "gpg:sign-and-deploy-file",
			"-settings=build/mvn.settings", "-Durl="+repo, "-DrepositoryId=ossrh",
			"-DpomFile="+meta.Package+".pom", "-Dfile="+meta.Package+".aar")
	}
}

func gomobileTool(subcmd string, args ...string) *exec.Cmd {
	cmd := exec.Command(filepath.Join(GOBIN, "gomobile"), subcmd)
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = []string{
		"GOPATH=" + build.GOPATH(),
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}

type mavenMetadata struct {
	Version      string
	Package      string
	Develop      bool
	Contributors []mavenContributor
}

type mavenContributor struct {
	Name  string
	Email string
}

func newMavenMetadata(env build.Environment) mavenMetadata {
//
	contribs := []mavenContributor{}
	if authors, err := os.Open("AUTHORS"); err == nil {
		defer authors.Close()

		scanner := bufio.NewScanner(authors)
		for scanner.Scan() {
//
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line[0] == '#' {
				continue
			}
//
			re := regexp.MustCompile("([^<]+) <(.+)>")
			parts := re.FindStringSubmatch(line)
			if len(parts) == 3 {
				contribs = append(contribs, mavenContributor{Name: parts[1], Email: parts[2]})
			}
		}
	}
//
	version := build.VERSION()
	if isUnstableBuild(env) {
		version += "-SNAPSHOT"
	}
	return mavenMetadata{
		Version:      version,
		Package:      "gbgm-" + version,
		Develop:      isUnstableBuild(env),
		Contributors: contribs,
	}
}

//

func doXCodeFramework(cmdline []string) {
	var (
		local  = flag.Bool("local", false, `Flag whbgmchain we're only doing a local build (skip Maven artifacts)`)
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. IOS_SIGNING_KEY)`)
		deploy = flag.String("deploy", "", `Destination to deploy the archive (usually "trunk")`)
		upload = flag.String("upload", "", `Destination to upload the archives (usually "gbgmstore/builds")`)
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

//
	build.MustRun(goTool("get", "golang.org/x/mobile/cmd/gomobile"))
	build.MustRun(gomobileTool("init"))
	bind := gomobileTool("bind", "--target", "ios", "--tags", "ios", "-v", "github.com/5sWind/bgmchain/mobile")

	if *local {
//
		bind.Dir, _ = filepath.Abs(GOBIN)
		build.MustRun(bind)
		return
	}
	archive := "gbgm-" + archiveBasename("ios", env)
	if err := os.Mkdir(archive, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	bind.Dir, _ = filepath.Abs(archive)
	build.MustRun(bind)
	build.MustRunCommand("tar", "-zcvf", archive+".tar.gz", archive)

//
	maybeSkipArchive(env)

//
	if err := archiveUpload(archive+".tar.gz", *upload, *signer); err != nil {
		log.Fatal(err)
	}
//
	if *deploy != "" {
		meta := newPodMetadata(env, archive)
		build.Render("build/pod.podspec", "Gbgm.podspec", 0755, meta)
		build.MustRunCommand("pod", *deploy, "push", "Gbgm.podspec", "--allow-warnings", "--verbose")
	}
}

type podMetadata struct {
	Version      string
	Commit       string
	Archive      string
	Contributors []podContributor
}

type podContributor struct {
	Name  string
	Email string
}

func newPodMetadata(env build.Environment, archive string) podMetadata {
//
	contribs := []podContributor{}
	if authors, err := os.Open("AUTHORS"); err == nil {
		defer authors.Close()

		scanner := bufio.NewScanner(authors)
		for scanner.Scan() {
//
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line[0] == '#' {
				continue
			}
//
			re := regexp.MustCompile("([^<]+) <(.+)>")
			parts := re.FindStringSubmatch(line)
			if len(parts) == 3 {
				contribs = append(contribs, podContributor{Name: parts[1], Email: parts[2]})
			}
		}
	}
	version := build.VERSION()
	if isUnstableBuild(env) {
		version += "-unstable." + env.Buildnum
	}
	return podMetadata{
		Archive:      archive,
		Version:      version,
		Commit:       env.Commit,
		Contributors: contribs,
	}
}

//

func doXgo(cmdline []string) {
	var (
		alltools = flag.Bool("alltools", false, `Flag whbgmchain we're building all known tools, or only on in particular`)
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

//
	gogetxgo := goTool("get", "github.com/karalabe/xgo")
	build.MustRun(gogetxgo)

//
	args := append(buildFlags(env), flag.Args()...)

	if *alltools {
		args = append(args, []string{"--dest", GOBIN}...)
		for _, res := range allToolsArchiveFiles {
			if strings.HasPrefix(res, GOBIN) {
//
				args = append(args, "./"+filepath.Join("cmd", filepath.Base(res)))
				xgo := xgoTool(args)
				build.MustRun(xgo)
				args = args[:len(args)-1]
			}
		}
		return
	}
//
	path := args[len(args)-1]
	args = append(args[:len(args)-1], []string{"--dest", GOBIN, path}...)

	xgo := xgoTool(args)
	build.MustRun(xgo)
}

func xgoTool(args []string) *exec.Cmd {
	cmd := exec.Command(filepath.Join(GOBIN, "xgo"), args...)
	cmd.Env = []string{
		"GOPATH=" + build.GOPATH(),
		"GOBIN=" + GOBIN,
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") || strings.HasPrefix(e, "GOBIN=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}

//

func doPurge(cmdline []string) {
	var (
		store = flag.String("store", "", `Destination from where to purge archives (usually "gbgmstore/builds")`)
		limit = flag.Int("days", 30, `Age threshold above which to delete unstalbe archives`)
	)
	flag.CommandLine.Parse(cmdline)

	if env := build.Env(); !env.IsCronJob {
		log.Printf("skipping because not a cron job")
		os.Exit(0)
	}
//
	auth := build.AzureBlobstoreConfig{
		Account:   strings.Split(*store, "/")[0],
		Token:     os.Getenv("AZURE_BLOBSTORE_TOKEN"),
		Container: strings.SplitN(*store, "/", 2)[1],
	}
	blobs, err := build.AzureBlobstoreList(auth)
	if err != nil {
		log.Fatal(err)
	}
//
	for i := 0; i < len(blobs); i++ {
		if !strings.Contains(blobs[i].Name, "unstable") {
			blobs = append(blobs[:i], blobs[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(blobs); i++ {
		for j := i + 1; j < len(blobs); j++ {
			iTime, err := time.Parse(time.RFC1123, blobs[i].Properties.LastModified)
			if err != nil {
				log.Fatal(err)
			}
			jTime, err := time.Parse(time.RFC1123, blobs[j].Properties.LastModified)
			if err != nil {
				log.Fatal(err)
			}
			if iTime.After(jTime) {
				blobs[i], blobs[j] = blobs[j], blobs[i]
			}
		}
	}
//
	for i, blob := range blobs {
		timestamp, _ := time.Parse(time.RFC1123, blob.Properties.LastModified)
		if time.Since(timestamp) < time.Duration(*limit)*24*time.Hour {
			blobs = blobs[:i]
			break
		}
	}
//
	if err := build.AzureBlobstoreDelete(auth, blobs); err != nil {
		log.Fatal(err)
	}
}
