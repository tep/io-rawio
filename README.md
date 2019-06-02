
# rawio
`import "toolman.org/io/rawio"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
Package rawio provides utilities for performing cancellable I/O operations.

## Install

``` sh
  go get toolman.org/io/rawio
```

## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [func ExtractFD(x interface{}) (int, error)](#ExtractFD)
* [func IsEAGAIN(err error) bool](#IsEAGAIN)
* [type RawReader](#RawReader)
  * [func NewRawReader(fd int) (*RawReader, error)](#NewRawReader)
  * [func (r *RawReader) Close() error](#RawReader.Close)
  * [func (r *RawReader) Fd() int](#RawReader.Fd)
  * [func (r *RawReader) Read(p []byte) (int, error)](#RawReader.Read)
  * [func (r *RawReader) ReadContext(ctx context.Context, p []byte) (int, error)](#RawReader.ReadContext)


#### <a name="pkg-files">Package files</a>
[epoll.go](/src/toolman.org/io/rawio/epoll.go) [fd.go](/src/toolman.org/io/rawio/fd.go) [latch.go](/src/toolman.org/io/rawio/latch.go) [reader.go](/src/toolman.org/io/rawio/reader.go) 



## <a name="pkg-variables">Variables</a>
``` go
var ErrNoFD = errors.New("no manner to discern file descriptor from object")
```
ErrNoFD indicates that no file descriptor could be extracted.

``` go
var NotifySignal = syscall.SIGIO
```
NotifySignal is the signal that will be used to interrupt a blocked Read operation.



## <a name="ExtractFD">func</a> [ExtractFD](/src/target/fd.go?s=2206:2248#L44)
``` go
func ExtractFD(x interface{}) (int, error)
```
ExtractFD attempts to extract a usable file descriptor from x -- which must
be either an *os.File or an object with a method having a signature of
"File() (*os.File, error)", such as most implementations of net.Conn. If x
is something else, ErrNoFD is returned.

If x is an *os.File, its Fd method is called -- otherwise, an *os.File is
aquired from the aforementioned File method and its Fd method is called.
In either case, a file descriptor is acquired then duped and the duplicate
file descriptor is returned.

Prior to returning, Close is called on the *os.File used to acquire the
original file descriptor and, if x is not an *os.File but is an io.Closer,
x will also be closed. If any of these close operations result in an error,
the duped file descriptor is also closed and an error is returned.

On success, a usable file descriptor and a nil error are returned and the
object from which it was extracted will be closed and no longer usable.

Note: We dupe the file descriptor before returning it because os.File has
a finalizer routine which calls Close when the object is garbage collected.
This also closes the file descriptor returned by its Fd method.

**Deprecated:** With the advent of the syscall.RawConn interface introduced in
Go v1.12, ExtractFD is no longer necessary.


## <a name="IsEAGAIN">func</a> [IsEAGAIN](/src/target/epoll.go?s=2378:2407#L98)
``` go
func IsEAGAIN(err error) bool
```
IsEAGAIN returns true if err represents a system error indicating that an
operation should be tried again.




## <a name="RawReader">type</a> [RawReader](/src/target/reader.go?s=1095:1168#L20)
``` go
type RawReader struct {
    sync.RWMutex
    // contains filtered or unexported fields
}
```
A RawReader performs read operations that may be cancelled -- either by
closing the reader from a separate goroutine or, through its ReadContext
method, if the context is cancelled.







### <a name="NewRawReader">func</a> [NewRawReader](/src/target/reader.go?s=1277:1322#L29)
``` go
func NewRawReader(fd int) (*RawReader, error)
```
NewRawReader creates a new RawReader from fd or nil and an error if the
RawReader cannot be created.





### <a name="RawReader.Close">func</a> (\*RawReader) [Close](/src/target/reader.go?s=1577:1610#L40)
``` go
func (r *RawReader) Close() error
```
Close will close the RawReader.  Any concurrently blocked read operations
will return io.EOF.
Close implements io.Closer.




### <a name="RawReader.Fd">func</a> (\*RawReader) [Fd](/src/target/reader.go?s=1905:1933#L63)
``` go
func (r *RawReader) Fd() int
```
Fd returns the file descriptor assocated with this RawReader.




### <a name="RawReader.Read">func</a> (\*RawReader) [Read](/src/target/reader.go?s=2159:2206#L75)
``` go
func (r *RawReader) Read(p []byte) (int, error)
```
Read implements io.Reader. If Read is blocked and r is closed in a separate
goroutine, Read will return 0 and io.EOF.




### <a name="RawReader.ReadContext">func</a> (\*RawReader) [ReadContext](/src/target/reader.go?s=2398:2473#L81)
``` go
func (r *RawReader) ReadContext(ctx context.Context, p []byte) (int, error)
```
ReadContext behaves similar to Read but may also be cancelled through the
given context. If ctx is cancelled, then ReadContext will return ctx.Err().

