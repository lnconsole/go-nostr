package nip08

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

var (
	mentionsPattern = regexp.MustCompile(`\#\[(\d+)\]`)
	wrapPattern     = regexp.MustCompile(`\[[p|e]:([a-fA-F0-9]+)\]`)
)

func DecodeMentions(content string, tags nostr.Tags) string {
	var (
		eventTags  = map[string]string{}
		pubkeyTags = map[string]string{}
		matches    = mentionsPattern.FindAllStringSubmatch(content, -1)
	)
	for _, match := range matches {
		label := match[0]
		labelNum, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if labelNum > len(tags)-1 {
			continue
		}
		if len(tags[labelNum]) < 2 {
			continue
		}

		t := tags[labelNum]
		if t.Key() == "e" {
			eventTags[label] = strings.Join(t, " ")
		}
		if t.Key() == "p" {
			pubkeyTags[label] = strings.Join(t, " ")
		}
	}

	return mentionsPattern.ReplaceAllStringFunc(content, func(match string) string {
		if e, ok := eventTags[match]; ok {
			return eWrap(e)
		}
		if p, ok := pubkeyTags[match]; ok {
			return pWrap(p)
		}
		return match
	})
}

func EncodeMentions(content string, availableIdx *int) (string, nostr.Tags) {
	var (
		newTags     = nostr.Tags{}
		encoded_idx = map[string]int{}
		matches     = wrapPattern.FindAllStringSubmatch(content, -1)
	)
	for _, match := range matches {
		tagStr := unwrap(match[1])
		tag := strings.Fields(tagStr)
		if _, ok := encoded_idx[match[0]]; !ok {
			encoded_idx[match[0]] = *availableIdx
			newTags = append(newTags, tag)
		}
		*availableIdx += 1
	}

	ret := wrapPattern.ReplaceAllStringFunc(content, func(s string) string {
		if idx, ok := encoded_idx[s]; ok {
			return fmt.Sprintf("#[%d]", idx)
		}
		return s
	})

	return ret, newTags
}

func pWrap(pubkey string) string {
	return fmt.Sprintf("[p:%s]", hex.EncodeToString([]byte(pubkey)))
}

func eWrap(eventID string) string {
	return fmt.Sprintf("[e:%s]", hex.EncodeToString([]byte(eventID)))
}

func unwrap(wrapped string) string {
	bytes, err := hex.DecodeString(wrapped)
	if err != nil {
		return ""
	}
	return string(bytes)
}
