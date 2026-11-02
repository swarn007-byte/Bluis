package storage

type Storage struct {
	data map[string]string
}

func NewStorage() *Storage{
	return &Storage{
		data:make(map[string]string),
	}
}

//set
func (s*Storage)Set(key string , val string){
	s.data[key]=val
}
// get
func (s*Storage)Get(key string) (string, bool){
	val, exists := s.data[key]
	return val, exists
}
// delete
func (s*Storage)Delete(key string){
	delete(s.data, key)
}


