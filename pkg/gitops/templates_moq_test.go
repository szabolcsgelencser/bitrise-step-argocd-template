// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package gitops

import (
	"sync"
)

// Ensure, that renderAllFileserMock does implement renderAllFileser.
// If this is not the case, regenerate this file with moq.
var _ renderAllFileser = &renderAllFileserMock{}

// renderAllFileserMock is a mock implementation of renderAllFileser.
//
//     func TestSomethingThatUsesrenderAllFileser(t *testing.T) {
//
//         // make and configure a mocked renderAllFileser
//         mockedrenderAllFileser := &renderAllFileserMock{
//             renderAllFilesFunc: func() error {
// 	               panic("mock out the renderAllFiles method")
//             },
//         }
//
//         // use mockedrenderAllFileser in code that requires renderAllFileser
//         // and then make assertions.
//
//     }
type renderAllFileserMock struct {
	// renderAllFilesFunc mocks the renderAllFiles method.
	renderAllFilesFunc func() error

	// calls tracks calls to the methods.
	calls struct {
		// renderAllFiles holds details about calls to the renderAllFiles method.
		renderAllFiles []struct {
		}
	}
	lockrenderAllFiles sync.RWMutex
}

// renderAllFiles calls renderAllFilesFunc.
func (mock *renderAllFileserMock) renderAllFiles() error {
	if mock.renderAllFilesFunc == nil {
		panic("renderAllFileserMock.renderAllFilesFunc: method is nil but renderAllFileser.renderAllFiles was just called")
	}
	callInfo := struct {
	}{}
	mock.lockrenderAllFiles.Lock()
	mock.calls.renderAllFiles = append(mock.calls.renderAllFiles, callInfo)
	mock.lockrenderAllFiles.Unlock()
	return mock.renderAllFilesFunc()
}

// renderAllFilesCalls gets all the calls that were made to renderAllFiles.
// Check the length with:
//     len(mockedrenderAllFileser.renderAllFilesCalls())
func (mock *renderAllFileserMock) renderAllFilesCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockrenderAllFiles.RLock()
	calls = mock.calls.renderAllFiles
	mock.lockrenderAllFiles.RUnlock()
	return calls
}