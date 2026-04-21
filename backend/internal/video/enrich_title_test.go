package video_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	videokit "github.com/shanegleeson/beepbopboop/backend/internal/video"
)

func TestEnrichMetadata_FoldsSynonymsAndSuppressesGenericNoise(t *testing.T) {
	video := model.Video{
		Title:       "Amazing video clip of a puppy meeting a kitten",
		Description: "A cute dog and cat reunion with lots of adorable energy.",
		SourceDesc:  "An adorable canine and feline meet for the first time.",
		Labels:      []string{"wimp", "video", "clip"},
	}

	got := videokit.EnrichMetadata(video)

	wantLabels := []string{"dogs", "cats", "animals", "cute"}
	if !reflect.DeepEqual(got.Labels, wantLabels) {
		t.Fatalf("labels: got %#v want %#v", got.Labels, wantLabels)
	}
	if got.SourceDescription != video.SourceDesc {
		t.Fatalf("source description: got %q want %q", got.SourceDescription, video.SourceDesc)
	}
	for _, banned := range []string{"video", "clip", "amazing", "interesting", "awesome"} {
		for _, label := range got.Labels {
			if label == banned {
				t.Fatalf("expected %q to be suppressed from labels, got %#v", banned, got.Labels)
			}
		}
	}
}

func TestEnrichMetadata_SnapshotSamples(t *testing.T) {
	cases := []struct {
		name string
		in   model.Video
		want videokit.Enrichment
	}{
		{
			name: "beatles bloopers",
			in: model.Video{
				Title:       "A blooper reel of Beatles recordings",
				Description: "Studio chatter, jokes, and rough takes from Beatles sessions.",
				SourceDesc:  "A collection of studio chatter and rough takes from Beatles recording sessions.",
				Labels:      []string{"wimp"},
			},
			want: videokit.Enrichment{
				Labels:            []string{"music", "behind-the-scenes", "nostalgia"},
				SourceDescription: "A collection of studio chatter and rough takes from Beatles recording sessions.",
				NormalizedTitle:   "A blooper reel of Beatles recordings",
			},
		},
		{
			name: "flying bike",
			in: model.Video{
				Title:       "Flying bike completes its first test flight.",
				Description: "A prototype hoverbike gets off the ground during an early demo.",
				SourceDesc:  "An early hoverbike prototype completes a test flight.",
				Labels:      []string{"wimp"},
			},
			want: videokit.Enrichment{
				Labels:            []string{"engineering", "vehicles", "innovation"},
				SourceDescription: "An early hoverbike prototype completes a test flight.",
				NormalizedTitle:   "Flying bike completes its first test flight",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := videokit.EnrichMetadata(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("snapshot mismatch:\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestGenerateTitle_AppliesGuardrailsAndFallbacks(t *testing.T) {
	enrichment := videokit.Enrichment{
		Labels:            []string{"music", "behind-the-scenes", "nostalgia"},
		SourceDescription: "A collection of studio chatter and rough takes from Beatles recording sessions.",
		NormalizedTitle:   "A blooper reel of Beatles recordings",
	}

	title := videokit.GenerateTitle(model.Video{
		Title:      "A blooper reel of Beatles recordings",
		SourceDesc: enrichment.SourceDescription,
	}, enrichment)

	if title == "" {
		t.Fatal("expected non-empty title")
	}
	if len(title) > 80 {
		t.Fatalf("expected title <= 80 chars, got %d: %q", len(title), title)
	}
	for _, banned := range []string{"shocking", "unbelievable", "you won't believe", "insane", "literally"} {
		if strings.Contains(strings.ToLower(title), banned) {
			t.Fatalf("title contains banned pattern %q: %q", banned, title)
		}
	}
	if title == "A blooper reel of Beatles recordings" {
		t.Fatalf("expected title generator to improve bland/raw title, got %q", title)
	}
}

func TestGenerateTitle_SnapshotSamples(t *testing.T) {
	cases := []struct {
		name string
		in   model.Video
		en   videokit.Enrichment
		want string
	}{
		{
			name: "beatles",
			in: model.Video{
				Title:      "A blooper reel of Beatles recordings",
				SourceDesc: "A collection of studio chatter and rough takes from Beatles recording sessions.",
			},
			en: videokit.Enrichment{
				Labels:            []string{"music", "behind-the-scenes", "nostalgia"},
				SourceDescription: "A collection of studio chatter and rough takes from Beatles recording sessions.",
				NormalizedTitle:   "A blooper reel of Beatles recordings",
			},
			want: "Beatles studio bloopers you probably haven't heard",
		},
		{
			name: "puppy kitten",
			in: model.Video{
				Title:      "Amazing video clip of a puppy meeting a kitten",
				SourceDesc: "An adorable canine and feline meet for the first time.",
			},
			en: videokit.Enrichment{
				Labels:            []string{"dogs", "cats", "animals", "cute"},
				SourceDescription: "An adorable canine and feline meet for the first time.",
				NormalizedTitle:   "A puppy meeting a kitten",
			},
			want: "A puppy and kitten meeting for the first time",
		},
		{
			name: "fallback cleaned title",
			in: model.Video{
				Title:      "Flying bike completes its first test flight.",
				SourceDesc: "An early hoverbike prototype completes a test flight.",
			},
			en: videokit.Enrichment{
				Labels:            []string{"engineering", "vehicles", "innovation"},
				SourceDescription: "An early hoverbike prototype completes a test flight.",
				NormalizedTitle:   "Flying bike completes its first test flight",
			},
			want: "Flying bike completes its first test flight",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := videokit.GenerateTitle(tc.in, tc.en); got != tc.want {
				t.Fatalf("title: got %q want %q", got, tc.want)
			}
		})
	}
}
