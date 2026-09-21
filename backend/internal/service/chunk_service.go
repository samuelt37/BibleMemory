package service

import "strings"

func chunkText(text string, targetWords int) []string {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")

	var chunks []string
	var current []string
	wordCount := 0

	flush := func() {
		if len(current) > 0 {
			chunks = append(chunks, strings.Join(current, "\n\n"))
			current = nil
			wordCount = 0
		}
	}

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		words := strings.Fields(p)

		// If a single paragraph alone exceeds the target, force-split it by word count.
		if len(words) > targetWords {
			flush() // whatever was building before, finalize it first
			for i := 0; i < len(words); i += targetWords {
				end := i + targetWords
				if end > len(words) {
					end = len(words)
				}
				chunks = append(chunks, strings.Join(words[i:end], " "))
			}
			continue
		}

		if wordCount+len(words) > targetWords && len(current) > 0 {
			flush()
		}

		current = append(current, p)
		wordCount += len(words)
	}

	flush()

	return chunks
}
