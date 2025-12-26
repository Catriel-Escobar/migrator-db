package migrate

type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}