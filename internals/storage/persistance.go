package storage

import(
	"os"
	"encoding/gob"
)

func (s *Storage) SaveToFile(filename string)error{ //filename - btati hai data kiss file mei save krna hai
	s.mu.RLock()
	defer s.mu.RUnlock()

	file,err:=os.Create(filename)
	if err!=nil{
		return err
	}
	defer file.Close()
  //go ek encoding package hai jo mechanism provide krta hai to convert go data structs to binary format	
	encoder:=gob.NewEncoder(file)
	if err:=encoder.Encode(s.data);err!=nil{
		return err
	}
	return nil
}
 

func (s *Storage) LoadFromFile(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&s.data); err != nil {
		return err
	}
	return nil
}