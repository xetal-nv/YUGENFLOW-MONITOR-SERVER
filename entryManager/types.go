package entryManager

type gateflow struct {
	in  int
	out int
}

// entry data
type entryData struct {
	name  string              // sensor number
	ts    int64               // timestamp
	count int                 // data received
	flows map[string]gateflow // floes at each gate
}
