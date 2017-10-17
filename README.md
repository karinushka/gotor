# gotor
CLI tool for managing RTorrent via its SCGI socket.

## Documentation
[https://godoc.org/github.com/karinushka/gotor](https://godoc.org/github.com/karinushka/gotor)

## Features
- Search and select all torrents by age, size and link count.
- Selected torrents can be stopped or deleted.

## Installation

To install the command line utility, run `go install "github.com/karinushka/gotor"`

## Command Line Utility

Here is a short description of command line usage.

`go-rtorrent`

```
NAME:
   rTorrent SCGI CLI - gotor

USAGE:
   gotor [options]

VERSION:
   1.0.0

AUTHOR(S):
   karinushka@github.com

OPTIONS:
   Following options control selection of the torrents:

      name      Select by specified regexp pattern in torrent name.
      newer     Select torrents newer than <int> days.
      nolinks   Select torrents which do not have any outgoing hardlinks.
      older     Select torrents older than <int> days.
      size      Select torrents larger than <int> megabytes.
   
   Options controlling the listing of selected torrents:

      sort      Sort listing by [name, size, date]. (default is date)
      verbose   Verbose output for listings.

   Options controlling the operation on selected torrents:

      stop      Stop matched torrents. The torrents are stopped and their file
                listing is printed out for manual examination.
      delete    Deletes the matched torrents and their files. This option
                should be provided together with "-stopped".

   Miscelaneous options:
   
      socket    RTorrent SCGI socket. (default /tmp/rtorrent.sock)

EXAMPLE USAGE:

   Following selects all torrents which are older than 100 days and larger than
   1024 megabytes:

      gotor -older 100 -size 1024 -verbose

   Once the listing of selected files is reviewed, they can be stopped and
   deleted:

      gotor -older 100 -size 1024 -stop -delete

```
