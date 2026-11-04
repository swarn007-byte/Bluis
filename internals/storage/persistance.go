package storage

import(
	"bytes"
	"os"
	"encoding/gob"
	"time"
)

type snapshot struct {
	Data map[string]string
	Expires map[string]time.Time
}

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
	//if err:=encoder.Encode(s.data);err!=nil{
	if err:=encoder.Encode(snapshot{Data:s.data,Expires:s.expires});err!=nil{
		return err
	}
	return nil
}
 

func (s *Storage) LoadFromFile(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	//file, err := os.Open(filename)
	//if err != nil {
	//	return err
	//}
	//defer file.Close()

	b,err:=os.ReadFile(filename)
	if err != nil {
		return err
	}

	//decoder := gob.NewDecoder(file)
	//if err := decoder.Decode(&s.data); err != nil {
	//	return err
	//}

	var snap snapshot
	if err:=gob.NewDecoder(bytes.NewReader(b)).Decode(&snap);err==nil{
		if snap.Data==nil{
			snap.Data=make(map[string]string)
		}
		if snap.Expires==nil{
			snap.Expires=make(map[string]time.Time)
		}
		s.data=snap.Data
		s.expires=snap.Expires
		return nil
	}

	var legacy map[string]string
	if err:=gob.NewDecoder(bytes.NewReader(b)).Decode(&legacy);err!=nil{
		return err
	}
	s.data=legacy
	s.expires=make(map[string]time.Time)
	return nil
}