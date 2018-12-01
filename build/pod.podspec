Pod::Spec.new do |spec|
  spec.name         = 'Gbgm'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/bgmchain/go-bgmchain'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Bgmchain Client'
  spec.source       = { :git => 'https://github.com/bgmchain/go-bgmchain.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gbgm.framework'

	spec.prepare_command = <<-CMD
    curl https://gbgmstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gbgm.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
