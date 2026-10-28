// Package profile installs the lok Basic macro into a LibreOffice user profile.
//
// LibreOfficeKit cannot change a document's page geometry through a UNO
// dispatch, so geometry is applied by a Basic macro run via runMacro. The macro
// lives in the profile's Standard Basic library.
//
// LibreOffice regenerates the Standard library the first time a fresh profile
// initializes Basic, which would erase a pre-written macro. Install therefore
// first establishes the profile with a throwaway soffice run, then adds the
// macro to the established library, leaving the default Module1 intact.
package profile
