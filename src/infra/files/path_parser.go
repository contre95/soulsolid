package files

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gosimple/unidecode"
)

// TemplatePathParser is an implementation of the PathParser interface that uses templates.
type TemplatePathParser struct {
	config *config.Manager
}

// NewTemplatePathParser creates a new TemplatePathParser.
func NewTemplatePathParser(cfg *config.Manager) *TemplatePathParser {
	return &TemplatePathParser{config: cfg}
}

// RenderPath renders a path for a track based on templates in the config.
func (p *TemplatePathParser) RenderPath(track *music.Track) (string, error) {
	pathsCfg := p.config.Get().Import.PathOptions
	var pathTemplate string
	switch track.Album.Type {
	case "ep":
		pathTemplate = pathsCfg.AlbumEP
	case "single":
		pathTemplate = pathsCfg.AlbumSingle
	case "soundtrack":
		pathTemplate = pathsCfg.AlbumSoundtrack
	default:
		pathTemplate = pathsCfg.DefaultPath
	}

	return p.renderPathTemplate(pathTemplate, track)
}

func (p *TemplatePathParser) renderPathTemplate(template string, track *music.Track) (string, error) {
	var renderErr error
	// Regex to find functions like %asciify{...}
	reFunc := regexp.MustCompile(`%(\w+)\{([^}]+)\}`)
	// First, process functions
	rendered := reFunc.ReplaceAllStringFunc(template, func(raw string) string {
		parts := reFunc.FindStringSubmatch(raw)
		if len(parts) != 3 {
			return raw // Should not happen
		}
		funcName := parts[1]
		argTemplate := parts[2]

		// Render the argument part of the function
		argValue, err := p.renderValues(argTemplate, track)
		if err != nil {
			renderErr = err
			return "ERROR"
		}

		switch funcName {
		case "asciify":
			return unidecode.Unidecode(argValue)
		case "if":
			// Simple if: %if{condition,true_value,false_value}
			args := strings.Split(argValue, ",")
			if len(args) >= 2 {
				condition, err := p.renderValues(args[0], track)
				if err != nil {
					renderErr = err
					return "ERROR"
				}
				if condition != "" && condition != "0" && condition != "false" {
					val, err := p.renderValues(args[1], track)
					if err != nil {
						renderErr = err
						return "ERROR"
					}
					return val
				} else if len(args) > 2 {
					val, err := p.renderValues(args[2], track)
					if err != nil {
						renderErr = err
						return "ERROR"
					}
					return val
				}
			}
			return ""
		default:
			return raw // Unknown function
		}
	})
	if renderErr != nil {
		return "", renderErr
	}

	// Then, process the remaining values
	return p.renderValues(rendered, track)
}

func (p *TemplatePathParser) renderValues(template string, track *music.Track) (string, error) {
	// Regex to find placeholders like $albumartist
	reVal := regexp.MustCompile(`\$(\w+)`)
	rendered := reVal.ReplaceAllStringFunc(template, func(raw string) string {
		var val string
		key := strings.TrimPrefix(raw, "$")
		switch key {
		case "albumartist":
			if len(track.Album.Artists) > 0 {
				val = track.Album.Artists[0].Artist.Name
			}
		case "album":
			val = track.Album.Title
		case "year":
			val = strconv.Itoa(track.Metadata.Year)
		case "original_year":
			val = strconv.Itoa(track.Metadata.OriginalYear)
		case "track":
			val = fmt.Sprintf("%02d", track.Metadata.TrackNumber)
		case "title":
			val = track.Title
		default:
			return raw // Unknown placeholder
		}
		// Sanitize path separators
		val = strings.ReplaceAll(val, "/", "-")
		return val
	})
	return rendered, nil
}
