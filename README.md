# go-page-process

## Purpose

go-page-process is a small static page renderer. It reads structured data from
`./data` (JSON, YAML, or TOML), merges in shared data from `./shared`, and
renders Handlebars templates found in `./.processor/templates` into an output
directory (default: `pages`).

### Expected project layout

```
.
├── .processor/templates/   # Handlebars templates (*.hbs)
├── data/                   # Page data (json/yaml/toml)
├── shared/                 # Optional shared data (json/yaml/toml)
└── pages/                  # Output (created)
```

Each data file should include a `_page` section to select a template and output
path:

```
{
  "_page": {
    "template": "blog/post",
    "output": "blog/hello.html"
  },
  "title": "Hello"
}
```

## Usage (binary)

Build and run the binary from the project root:

```
go build -o go-page-processor .
./go-page-processor --output-path pages
```

All paths can be absolute or relative. Defaults match the layout above. Use
`--root` to resolve relative paths from a single base directory.

Options:

- `-o, --output-path` (default: `pages`)
- `-d, --data-path` (default: `data`)
- `-s, --shared-path` (default: `shared`)
- `-t, --templates-path` (default: `.processor/templates`)
- `-r, --root` (default: current working directory)

Run `./go-page-processor --help` to see all flags.

The CLI exits non-zero if processing fails (for example, missing templates or
data parse errors).

Example using a custom root directory:

```
./go-page-processor --root /path/to/site --output-path build
```

## CLI options

All options have short and long forms:

- `-o, --output-path` - Output directory (default: `pages`)
- `-d, --data-path` - Data directory (default: `data`)
- `-s, --shared-path` - Shared data directory (default: `shared`)
- `-t, --templates-path` - Templates directory (default: `.processor/templates`)
- `-r, --root` - Base directory for resolving relative paths (default: current
  working directory)
- `-h, --help` - Display help

## Usage (API)

Embed the processor in another Go program:

```
import "github.com/andriyg76/go-page-process/processor"

config := processor.DefaultConfig()
config.OutputPath = "pages"
config.RootPath = "/path/to/site"
proc := processor.New(config)
if err := proc.Process(); err != nil {
  // handle error
}
```

## Data files (pages)

Page data files live under the data directory. Supported formats are JSON
(`.json`, `.json5`), YAML (`.yaml`, `.yml`), and TOML (`.toml`).

Each page file must contain a `_page` section:

```
{
  "_page": {
    "template": "blog/post",
    "output": "blog/hello.html"
  },
  "title": "Hello",
  "author": "Ada"
}
```

Notes:

- `template` is the name of the template file relative to the templates
  directory, without the `.hbs` extension.
- `output` is optional. If omitted, the output file name defaults to the input
  file name.
- If `output` starts with `/`, the leading slash is removed so the output stays
  within the output directory.
- Files without a `_page` section are ignored.

YAML example:

```
_page:
  template: blog/post
  output: blog/hello.html
title: Hello
author: Ada
```

TOML example:

```
[_page]
template = "blog/post"
output = "blog/hello.html"

title = "Hello"
author = "Ada"
```

## Shared data

Shared data files are loaded before page processing and are available under the
`_shared` key in templates and page data. The shared directory is scanned
recursively.

File names (without extension) become keys, and nested directories become nested
objects.

Example structure:

```
shared/
  site.json
  navigation/
    menu.yaml
```

Example content:

`shared/site.json`

```
{ "name": "Docs", "url": "https://example.com" }
```

`shared/navigation/menu.yaml`

```
items:
  - label: Home
    href: /
```

Access in a template:

```
<title>{{_shared.site.name}}</title>
<nav>{{#each _shared.navigation.menu.items}}<a href="{{href}}">{{label}}</a>{{/each}}</nav>
```

To generate shared data before processing, create or update files in the shared
directory as part of your build pipeline before running the processor.

## Usage (Docker)

Build the image:

```
docker build -t go-page-processor .
```

Run it by mounting the project directory into `/data`:

```
docker run --rm -v "$PWD":/data go-page-processor --output-path pages
```

The container runs with `WORKDIR /data`, so `data/`, `shared/`, and
`.processor/templates/` should exist under the mounted directory.