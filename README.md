This app is the go version of the publish-failure-resolver (reImportContent.sh) bash script.
It makes use of concurrency, instead of republishing each content sequentially.

It can be used for platform load testing as well.

# Usage

`go run app.go -uuids <uuids-file-path> -post <cms-notifier-endpoint> -read <native-store-endpoint>`

The uuids file format could be a JSON array with uuids or a text file with each UUID on a new line.
