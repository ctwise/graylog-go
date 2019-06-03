# Graylog-Go

Command-line interface to search and interrogate a Graylog instance. Very useful for searching and tailing logs from the command-line.

Originally came from https://github.com/bvargo/gtail. I converted it first to Python 3, then Go.

```text
usage: graylog [-h|--help] [--list-streams] [-a|--application "<value>"]
               [-q|--query "<value>"] [-e|--export "<value>"] [-l|--limit
               <integer>] [-s|--stream "<value>"] [-t|--tail] [-c|--config
               "<value>"] [-r|--range "<value>"] [--start "<value>"] [--end
               "<value>"] [-j|--json] [--no-colors]

               Search and tail logs from Graylog.

Arguments:

  -h  --help          Print help information
      --list-streams  List Graylog streams and exit.
  -a  --application   Special case to search the 'application' message field,
                      e.g., -a send-email is equivalent to -q
                      'application:send-email'. Merged with the -q query using
                      'AND' if the -q query is present.
  -q  --query         Query terms to search on (Elasticsearch syntax). Defaults
                      to '*'.
  -e  --export        Export specified fields as CSV into a file named
                      'export.csv'. Format is 'field1,field2,field3...'.
                      Requires --start (and, optionally, --end) option.
  -l  --limit         The maximum number of messages to request from Graylog.
                      Must be greater then 0. Default: 300
  -s  --stream        The name of the stream(s) to display messages from.
                      Default: all streams.
  -t  --tail          Whether to tail the output. Requires a relative search.
  -c  --config        Path to the config file. Default: /Users/ctwise/.graylog
  -r  --range         Time range to search backwards from the current moment.
                      Examples: 30m, 2h, 4d. Default: 2h
      --start         Starting time to search from. Allows variable formats,
                      including '1:32pm' or '1/4/2019 12:30:00'.
      --end           Ending time to search from. Allows variable formats,
                      including '6:45am' or '2019-01-04 12:30:00'. Defaults to
                      now if --start is provided but no --end.
  -j  --json          Output messages in json format. Shows the modified log
                      message, not the untouched message from Graylog. Useful
                      in understanding the fields available when creating
                      Format templates or for further processing.
      --no-colors     Don't use colors in output.
```

Requires a configuration file be setup. By default, the application looks in ~/.graylog.

A default configuration file might look like:

```ini
[server]
; Graylog REST API
uri: https://<server>:<port>/api
; optional username and password
username: <username>
password: <password>
ignoreCert: false
[formats]
; log formats (list them most specific to least specific, they will be tried in order)
; all fields must be present or the format won't be applied
; Formats use the Go template syntax.
;
; access log w/bytes
format1: <{{.source}}> {{.client_ip}} {{.ident}} {{.auth}} [{{.apache_timestamp}}] "{{.method}} {{.request_page}} HTTP/{{.http_version}}" {{.server_response}} {{.bytes}}
; access log w/o bytes
format2: <{{.source}}> {{.client_ip}} {{.ident}} {{.auth}} [{{.apache_timestamp}}] "{{.method}} {{.request_page}} HTTP/{{.http_version}}" {{.server_response}}
; java log entry
format3: <{{.source}}> {{._long_time_timestamp}} {{._level_color}}{{printf "%-5.5s" .loglevel}}{{._reset}} {{printf "%-20.20s" ._short_classname}} : {{._message_text}}
; syslog
format4: <{{.source}}> {{._long_time_timestamp}} {{._level_color}}{{printf "%-5.5s" .loglevel}}{{._reset}} [{{.facility}}] : {{._message_text}}
; generic entry with a loglevel
format5: <{{.source}}> {{._long_time_timestamp}} {{._level_color}}{{printf "%-5.5s" .loglevel}}{{._reset}} : {{._message_text}}
```
