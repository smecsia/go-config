package mock

import "github.com/stretchr/testify/mock"

type GitMock struct {
	mock.Mock
}

func (m *GitMock) CommitAndPush(msg string) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *GitMock) Root() string {
	args := m.Called()
	return args.Get(0).(string)
}

func (m *GitMock) Hash() (string, error) {
	args := m.Called()
	return args.Get(0).(string), args.Error(1)
}

func (m *GitMock) HashShort() (string, error) {
	args := m.Called()
	return args.Get(0).(string), args.Error(1)
}

func (m *GitMock) IsWorkTreeClean() (bool, string, error) {
	args := m.Called()
	return args.Get(0).(bool), args.Get(1).(string), args.Error(2)
}
