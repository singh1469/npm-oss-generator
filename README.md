# OSS Generator for NPM 

## Purpose
Produce a JSON blob of dependencies and learning activity to Golang.

## Usage
```sh
$ cd /my/play/area
$ cd git clone <url>
$ go run main.go -path=/path/to/node/modules/directory
```
Produces a JSON blob of dependencies to stdout.

## Additional Options
| Option     | Syntax | Description |
| ---      | ---       | ---
| onlyDirectDependencies | `-onlyDirectDependencies`         | Fetch direct dependencies only |
| onlyDirectDevDependencies     | `-onlyDirectDevDependencies`        | Fetch direct dev dependencies only
| output     | `-output=file`        | Redirect stdout to a file |

## Release ready?
I won't be creating a release until test coverage and docs are in place.

#License
MIT