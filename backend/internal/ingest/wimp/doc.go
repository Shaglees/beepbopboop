// Package wimp is the ingest adapter that converts archived wimp.com pages
// (Wayback Machine HTML captures) into catalog candidates ready for the video
// catalog (see repository.VideoRepo).
//
// Why Wayback and not the Wimp CDN:
// The Wimp self-hosted video binaries (cdn.wimp.com, cdn2.wimp.com) were
// never archived by Wayback's crawler (confirmed empirically: every CDX
// query for those hosts and for *.mp4 under wimp.com returns zero captures).
// Playable byte recovery from the Wimp archive is therefore impossible.
//
// What Wayback DOES have is the wimp.com HTML pages, extensively captured
// from 2005 onwards. For post-Flash-era pages (roughly 2015+), those pages
// embed YouTube or Vimeo iframes directly; those third-party videos are
// still playable today.
//
// So this adapter treats Wimp as a curation index, not a video store:
//
//  1. Look up the newest HTTP 200 HTML capture of a given wimp.com page via
//     the Wayback CDX API.
//  2. Fetch the capture in id_-form (no Wayback toolbar rewrite).
//  3. Parse title, description, og:image, og:url, keywords.
//  4. Extract the first embedded YouTube or Vimeo reference as the candidate's
//     provider (watch_url / embed_url).
//  5. Return a populated model.Video. If no third-party embed is found,
//     return ErrNoLiveEmbed and let the caller decide whether to drop the
//     candidate or queue it for a later title-match fallback.
//
// The adapter does not write to the database or crawl an index of URLs;
// those are the jobs of the caller (see issue #160 and #172 for scope).
package wimp
