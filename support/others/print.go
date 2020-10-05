package others

import (
	"encoding/json"
	"fmt"
)

func PrettyPrint(data interface{}) {
	s, _ := json.MarshalIndent(data, "", "\t")
	fmt.Println(string(s))
}
