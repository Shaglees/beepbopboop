package wimp

import "errors"

// ErrNoCapture means the Wayback CDX API returned no HTTP 200 HTML captures
// for the requested URL. Caller may retry with a different URL form
// (http vs https, trailing slash) or drop the candidate.
var ErrNoCapture = errors.New("wimp: no wayback capture for url")

// ErrNoLiveEmbed means the archived HTML contained no recognizable
// YouTube or Vimeo reference. The page's metadata may still be useful for
// a future title-match fallback; that is out of scope for this adapter.
var ErrNoLiveEmbed = errors.New("wimp: no live third-party embed in archived html")
