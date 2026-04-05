package filereadtool

import "errors"

// ErrImageProcessorDeferred mirrors imageProcessor.ts lazy Sharp/native loading; headless Read returns this for image paths until Phase 6 follow-on.
var ErrImageProcessorDeferred = errors.New("filereadtool: image read deferred (imageProcessor.ts / sharp)")
