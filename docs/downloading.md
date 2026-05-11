# Downloading

Soulsolid supports downloading music through pluggable downloaders. The system is designed to be agnostic to specific music services, allowing developers to create plugins for different platforms.

## Plugin Architecture

Downloaders are implemented as Go plugins that can be loaded at runtime. Each plugin:

- Implements the `Downloader` interface
- Handles authentication and API communication
- Manages quality selection and format preferences
- Provides user information and status reporting

## Configuration

Downloaders are configured in the `config.yaml` file:

```yaml
downloaders:
  plugins:
    - name: "examplefy"
      path: "/path/to/examplefy.so"
      # url: "https://github.com/user/soulsolid-examplefy"  # alternative: build from git
      icon: "https://example.com/icon.png"
      config:
        api_key: "your_api_key"
  artwork:
    embedded:
      enabled: true
      size: 1000   # max dimension in pixels
      quality: 85  # JPEG quality (0-100)
```

## Downloading Process

The download process varies by plugin implementation, but generally follows this pattern:

1. **Authentication**: Plugin handles authentication with the music service
2. **Metadata Retrieval**: Fetches track/album metadata from the service's API
3. **Audio Download**: Downloads audio data in the requested format
4. **Format Processing**: Handles any decryption or format conversion if needed
5. **File Writing**: Saves the audio file to the configured download directory

## Tagging Process

After downloading, Soulsolid embeds comprehensive metadata into the audio files:

### For MP3 files (ID3v2 tags):

• Title, Artist, Album, Year, Genre
• Track/Position in set, Disc number
• ISRC, BPM, ReplayGain
• Album art (downloaded from the service)
• Publisher, Barcode, Composer, Lyrics

### For FLAC files (Vorbis comments):

• Similar metadata fields as MP3
• Additional Vorbis-specific fields like VERSION, DISCNUMBER

## Creating Custom Plugins

See the [Plugin Development Guide](plugins.md) for information on creating custom downloader plugins.
