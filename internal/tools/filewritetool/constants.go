package filewritetool

// ErrFileUnexpectedlyModified mirrors FileEditTool/constants.ts FILE_UNEXPECTEDLY_MODIFIED_ERROR.
const ErrFileUnexpectedlyModified = "File has been unexpectedly modified. Read it again before attempting to write it."

// ErrFileModifiedSinceRead mirrors FileWriteTool.ts validateInput (errorCode 3).
const ErrFileModifiedSinceRead = "File has been modified since read, either by the user or by a linter. Read it again before attempting to write it."
