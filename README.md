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

Run `./go-page-processor --help` to see all flags.

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