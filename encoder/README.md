#### Encoder.
This is a simple wrapper to the json.Marshal, which adds ability to skip some fields
of structure.  
Unlike 'render' package it doesn't write anything, just returns marshalled data.  
It's useful for things like passwords, statuses, activation codes, etc... 

E.g.:

```go
type Some struct {
	Login    string        `json:"login"`
	Password string        `json:"password,omitempty"  out:"false"`
}
```

Field 'Password' won't be exported.

