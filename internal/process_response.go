package internal

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

func (m *Manager) parseAIResponse(response string) (AIResponse, error) {
	// Tag mapping: tag name -> field
	type tagInfo struct {
		name     string
		isArray  bool
		isBool   bool
		setField func(*AIResponse, string)
	}
	tags := []tagInfo{
		// All action tags are now handled separately below
		{"RequestAccomplished", false, true, func(r *AIResponse, v string) { r.RequestAccomplished = isTrue(v) }},
		{"ExecPaneSeemsBusy", false, true, func(r *AIResponse, v string) { r.ExecPaneSeemsBusy = isTrue(v) }},
		{"WaitingForUserResponse", false, true, func(r *AIResponse, v string) { r.WaitingForUserResponse = isTrue(v) }},
		{"NoComment", false, true, func(r *AIResponse, v string) { r.NoComment = isTrue(v) }},
		{"CreateExecPane", false, true, func(r *AIResponse, v string) { r.CreateExecPane = isTrue(v) }},
	}

	clean := response
	r := AIResponse{}
	cleanForMsg := clean

	// Generic regex for tags with optional pane_id
	reWithPaneID := func(tagName string) *regexp.Regexp {
		return regexp.MustCompile(fmt.Sprintf(`(?s)<%s(?: pane_id="([^"]*)")?>(.*?)</%s>`, tagName, tagName))
	}

	// Handle ExecCommand
	reExec := regexp.MustCompile(`(?s)<ExecCommand([^>]*)>(.*?)</ExecCommand>`)
	execMatches := reExec.FindAllStringSubmatch(clean, -1)
	for _, match := range execMatches {
		if len(match) >= 3 {
			attrs := match[1]
			command := html.UnescapeString(strings.TrimSpace(match[2]))

			rePaneID := regexp.MustCompile(`pane_id="([^"]*)"`)
			reWait := regexp.MustCompile(`wait="(true|false|1|0)"`)

			paneIDMatch := rePaneID.FindStringSubmatch(attrs)
			waitMatch := reWait.FindStringSubmatch(attrs)

			var paneID string
			if len(paneIDMatch) > 1 {
				paneID = paneIDMatch[1]
			}
			var wait bool
			if len(waitMatch) > 1 {
				wait = isTrue(waitMatch[1])
			}

			r.ExecCommand = append(r.ExecCommand, ExecCommandInfo{PaneID: paneID, Wait: wait, Command: command})
		}
	}
	// Use a more general regex for cleaning to handle any attribute order.
	cleanForMsg = regexp.MustCompile(`(?s)<ExecCommand[^>]*>.*?</ExecCommand>`).ReplaceAllString(cleanForMsg, "")

	// Handle TmuxSendKeys
	reSendKeys := reWithPaneID("TmuxSendKeys")
	sendKeysMatches := reSendKeys.FindAllStringSubmatch(clean, -1)
	for _, match := range sendKeysMatches {
		if len(match) >= 3 {
			r.SendKeys = append(r.SendKeys, SendKeysInfo{PaneID: match[1], Keys: html.UnescapeString(match[2])})
		}
	}
	cleanForMsg = reSendKeys.ReplaceAllString(cleanForMsg, "")

	// Handle PasteMultilineContent
	rePaste := reWithPaneID("PasteMultilineContent")
	pasteMatches := rePaste.FindAllStringSubmatch(clean, -1)
	for _, match := range pasteMatches {
		if len(match) >= 3 {
			r.PasteMultilineContent = append(r.PasteMultilineContent, PasteInfo{PaneID: match[1], Content: html.UnescapeString(strings.TrimSpace(match[2]))})
		}
	}
	cleanForMsg = rePaste.ReplaceAllString(cleanForMsg, "")

	// Handle ReadFile
	reReadFile := reWithPaneID("ReadFile")
	readFileMatches := reReadFile.FindAllStringSubmatch(clean, -1)
	for _, match := range readFileMatches {
		if len(match) >= 3 {
			paneID := match[1]
			filePathsStr := html.UnescapeString(strings.TrimSpace(match[2]))
			// Split by whitespace to handle multiple files in one tag
			filePaths := strings.Fields(filePathsStr)
			for _, fp := range filePaths {
				r.ReadFile = append(r.ReadFile, ReadFileInfo{PaneID: paneID, FilePath: fp})
			}
		}
	}
	cleanForMsg = reReadFile.ReplaceAllString(cleanForMsg, "")

	// Handle the simple boolean tags
	for _, t := range tags {
		reTag := regexp.MustCompile(fmt.Sprintf(`(?s)<%s>(.*?)</%s>`, t.name, t.name))
		tagMatches := reTag.FindAllStringSubmatch(clean, -1)
		for _, m := range tagMatches {
			if len(m) < 2 {
				continue
			}
			val := strings.TrimSpace(m[1])
			if !t.isBool {
				val = html.UnescapeString(val)
			}
			if t.isArray {
				t.setField(&r, val)
			} else {
				t.setField(&r, val)
			}
		}
		// For message: remove all tag blocks
		cleanForMsg = reTag.ReplaceAllString(cleanForMsg, "")
	}

	// Clean up empty ``` blocks that might be left over after tag removal.
	cleanForMsg = regexp.MustCompile("(?s)`{3}\\w*\\s*`{3}").ReplaceAllString(cleanForMsg, "")

	// Message: trim, collapse multiple newlines
	msg := strings.TrimSpace(cleanForMsg)
	msg = collapseBlankLines(msg)
	r.Message = msg

	return r, nil
}

// Helper: check if string is "1" or "true" (case-insensitive)
func isTrue(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true"
}

// Collapse multiple blank lines to a single newline
func collapseBlankLines(s string) string {
	return mustCompile(`\n{2,}`).ReplaceAllString(s, "\n")
}

// mustCompile is a helper for regexp.MustCompile
func mustCompile(expr string) *regexp.Regexp {
	re, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	return re
}
