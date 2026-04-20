# Path Parser Documentation

This document describes all available placeholders and functions for the Soulsolid path parser, which is used to organize music files into directories based on metadata.

## Placeholders

Placeholders are variables that get replaced with actual metadata values from the track. They are prefixed with a `$` symbol.

### Available Placeholders

| Placeholder | Description | Data Type | Example Value |
|-------------|-------------|-----------|---------------|
| `$albumartist` | The name of the first album artist | String | "The Beatles" |
| `$album` | The album title | String | "Abbey Road" |
| `$year` | The release year | Integer | "1969" |
| `$original_year` | The original release year (if different from release year) | Integer | "1969" |
| `$track` | The track number, zero-padded to 2 digits | String | "01" |
| `$title` | The track title | String | "Come Together" |
| `$format` | Audio format of the file | String | "flac" |
| `$genre` | Music genre | String | "Rock" |

## Functions

Functions perform operations on placeholders or other values. They are prefixed with `%` and use curly braces for arguments.

### Available Functions

#### artistfolder

Groups tracks into letter-based subfolders by artist name, similar to how music stores organize physical media. The value is first asciified, then prefixed with its uppercase first letter (A–Z) or `#` for non-alphabetic names.

**Syntax:** `%artistfolder{value}`

**Parameters:**
- `value`: Any string value or placeholder (typically `$albumartist`)

**Examples:**
```
%artistfolder{$albumartist}
```
- "The Beatles" → `T/The Beatles`
- "Björk" → `B/Bjork`
- "2Pac" → `#/2Pac`

A typical path using `%artistfolder`:
```
%artistfolder{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```
**Result:** `T/The Beatles/Abbey Road (1969)/01 Come Together`

#### asciify

Converts text to ASCII by removing accents, diacritics, and special characters.

**Syntax:** `%asciify{value}`

**Parameters:**
- `value`: Any string value or placeholder

**Example:**
```
%asciify{$albumartist}
```
For "Björk" → "Bjork"

#### if

Conditional rendering that returns one of two values based on a condition.

**Syntax:** `%if{condition,true_value,false_value}`

**Parameters:**
- `condition`: Expression to evaluate (returns true if not empty, "0", or "false")
- `true_value`: Value to return if condition is true
- `false_value`: Value to return if condition is false (optional)

**Examples:**
```
%if{$original_year,$original_year,$year}
```
Returns `$original_year` if it exists, otherwise `$year`

```
%if{$albumartist,$albumartist,Various Artists}
```
Returns `$albumartist` if it exists, otherwise "Various Artists"

## Usage Examples

### Basic Path Template
```
$albumartist/$album ($year)/$track $title
```

**Result:** `The Beatles/Abbey Road (1969)/01 Come Together`

### With ASCII Conversion
```
%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

**Result:** `The Beatles/Abbey Road (1969)/01 Come Together`

### Album Type Specific Paths

#### Default Album
```
%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

#### EP
```
%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

#### Single
```
%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

#### Soundtrack
```
%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

#### Compilation
```
%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

## Advanced Examples

### Conditional Album Artist
```
%if{$albumartist,%asciify{$albumartist},Various Artists}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}
```

### Year with Fallback
```
%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,%if{$year,$year,Unknown}})/%asciify{$track $title}
```

## Notes

- All placeholders are replaced with actual metadata values from the track
- Functions can be nested within each other
- Unknown placeholders are left as-is in the output
- The `%asciify` function is particularly useful for creating filesystem-safe paths
- The `%if` function allows for conditional logic in path templates
- Track numbers are automatically zero-padded to 2 digits
- Years are converted to strings automatically

## Configuration

These templates can be configured in the `config.yaml` file under the `import.paths` section:

```yaml
import:
  paths:
    default_path: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:ep: '%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:single: '%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:soundtrack: '%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    compilations: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
```

## Reorganize Library

The **Reorganize** feature applies the current path templates to every track in the library and physically moves files to match. It is useful after changing path templates or after bulk metadata corrections.

### How It Works

1. Iterates all tracks in the library in batches of 100
2. Computes each track's desired path using the active path templates
3. Skips tracks whose files are missing on disk
4. Skips tracks already at the correct path
5. Moves the file to the new path, creating intermediate directories as needed
6. Updates the track's path in the database
7. Removes empty directories left behind after moving

Progress is tracked as a background job and can be monitored in the Jobs section.

### FAT32 Safe Mode

When **FAT32 Safe** is enabled, every path segment is additionally sanitized for FAT32 compatibility before the file is moved:

- Replaces forbidden characters (`: * ? " < > | \`) with `-`
- Strips trailing dots and spaces from each segment
- Truncates segments to 255 bytes at valid UTF-8 boundaries, preserving the file extension for the filename

This is useful when the library is stored on or synced to a FAT32 volume (e.g. an SD card or some NAS configurations).

### API

```
POST /analyze/reorganize
Content-Type: application/x-www-form-urlencoded

fat32_safe=true   # optional, defaults to false
```

The endpoint triggers a background job and returns a toast notification. The job result includes counts for `moved`, `skipped`, and `errors` out of the total track count.
