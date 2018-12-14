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

package gbgm

import (
	"errors"
	"fmt"
)

//
type Strings struct{ strs []string }

//
func (s *Strings) Size() int {
	return len(s.strs)
}

//
func (s *Strings) Get(index int) (str string, _ error) {
	if index < 0 || index >= len(s.strs) {
		return "", errors.New("index out of bounds")
	}
	return s.strs[index], nil
}

//
func (s *Strings) Set(index int, str string) error {
	if index < 0 || index >= len(s.strs) {
		return errors.New("index out of bounds")
	}
	s.strs[index] = str
	return nil
}

//
func (s *Strings) String() string {
	return fmt.Sprintf("%v", s.strs)
}
