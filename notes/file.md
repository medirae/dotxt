# description
## file/info
this is for having a glance at a file in terms of metadata
### watcher
it needs to register files based on their creation triggers or when starting up
it needs to be able to discern some logic so it needs these info:
    - `addr` and `destAddr` if it is a symlink
    - `isSymlink`
    - `isFile` or a directory
    - `exists` in filesystem or just theory
    - `hasRead`
    - `hasWrite`
since the filesystem must be presumed ever-changing to avoid conflicts,
    it needs to be able to refetch the info that it was keeping
since fetching these info is costly, their sources must be kept seperately and fetched only if necessary
since info is not necessary at all times it should be possible to make a shallow object

### internal events
to discern changes for a file, it would be costly to record a checksum of content, so a shallower checksum would that of size and modtime or something else too that's tangible (maybe path and also inode...)

### os info
os.Lstat(name string) (FileInfo, error)
os.Stat(name string) (FileInfo, error)
fs.FileInfo.Mode() FileMode
fs.FileMode.IsDir() bool
fs.FileMode.IsRegular() bool
fs.FileMode.Size() int64
fs.FileMode.ModTime() time.Time

filepath.EvalSymlinks(path string) (string, error)
os.ReadDir(name string) ([]os.DirEntry, error)
os.ReadLink(name string) (string, error)

## file/path
this is related to the app domain.
it should represent the app file structure; it should provide the basic functions of setting up the structure
it should present the struct that represents the idea of a task list; each one has to have methods that'll lead to related files of a list
it's not necessary to link this directly to file/info, the transistion will happen once a function is necessary to be called; from within there the addr of file/info struct will passed to create a file/path struct
it should provide parse functions for paths

# design solution
## file
```go
func Walk(root string, fn func(path string, info file/info.Info, err error)) error
func MkDir(path string, all bool) error
func MkStructure() error

func Create(path string) error
func Delete(path string) error
func Read(path string) ([]string, error)
func Write(text string, path string) error
func Append(text string, path string) error
func Rename(old, new *file/paths.Path) error
func Copy(src, dest string) error
func Backup(path *file/paths.Path) error
func Archive(path string) error
func List(path string) ([]*file/paths.Path, error)
```

## file/path
```go
const DefaultFileName
const FileDirName
const ETCDirName
const ArchiveDirName
func TextsDir() string
func ETCDir() string
func ArchiveDir() string
type Path struct {
    Name string // path relative to textsDir
    Path string // absolute path
}
func (p *Path) String() string
func (p *Path) Done() string
func (p *Path) Backup() string
// this returns the paths located in arhiveDir for the task, its done file, and its backup file respectively
func (p *Path) Archive() (string, string, string)
// takes a string and normalizes it into an absolute path under textsDir
func ParseFilepath(path string) (string, error)
// takes a string, parses it, makes it into Path struct, and evals its name // this is the main function of this module
func NewPath(path string) (*Path, error)
func ParseDirpath(path string) (string, error)
// creates the structure of the app
func MkDirs() error
```

## file/info
```go
type Info struct {
    Addr string
    DestAddr string
    IsSymlink bool
    IsFile bool
    Exists bool
    HasRead bool
    HasWrite bool
    Size int64
    ModTime time.Time
    lastProbed time.Time
    lock sync.RWMutex
}

// no errors if symlink target doesn't exist; only errors if the path is not symlink, nor regular file, nor a directory
func Identify(path string) (*Info, error)
// loops through symlinks if any to find a target; errors if no valid target
func EvalSymlink(path string) (string, os.FileInfo, error)
// checks permissions against uid and gid
func HasPerms(path string, info os.FileInfo) (bool, bool, os.FileInfo, error)
func NewInfo(path string) *Info
// returns an info in any case
func EnsureInfo(i *Info, path string, force bool) (*Info, error)
func (i *Info) NeedsRefresh() bool
func (i *Info) Refresh(force bool) error
func (i *Info) probe() error
```