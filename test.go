package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	proto "github.com/golang/protobuf/proto"
)

func main() {
	logger := log.New(os.Stdout, "", 0)

	hs := setup(logger)

	logger.Printf("Listening on http://0.0.0.0%s\n", hs.Addr)

	hs.ListenAndServe()
}

func setup(logger *log.Logger) *http.Server {
	return &http.Server{
		Addr:         getAddr(),
		Handler:      newServer(logWith(logger)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func getAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}

	return ":8383"
}

func newServer(options ...Option) *Server {
	s := &Server{logger: log.New(ioutil.Discard, "", 0)}

	for _, o := range options {
		o(s)
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/", s.index)

	return s
}

type Option func(*Server)

func logWith(logger *log.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

type Server struct {
	mux    *http.ServeMux
	logger *log.Logger
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	s.log("%s %s", r.Method, r.URL.Path)
	s.unmarshallData(body)

	s.mux.ServeHTTP(w, r)
}

func (s *Server) log(format string, v ...interface{}) {
	s.logger.Printf(format+"\n", v...)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, world!"))
}

func (s *Server) unmarshallData(data []byte) {
	deviceData := &DeviceDataArray{}
	err := proto.Unmarshal(data, deviceData)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}

	for _, dataItem := range deviceData.DeviceDataItems {

		guid := guidFromUint64(dataItem.DeviceId.Hi, dataItem.DeviceId.Lo)

		s.log(guid)
		// s.log("%x-%x-%x-%x-%x", highBytes[0:4], highBytes[4:6], highBytes[6:8], lowBytes[0:2], lowBytes[2:8])
		for _, dataValue := range dataItem.DeviceValues {
			s.log("%x - %f", dataValue.Obis, dataValue.Value)
		}
	}
}

func guidFromUint64(high, low uint64) string {
	lowBytes := make([]byte, 8)
	highBytes := make([]byte, 8)

	binary.LittleEndian.PutUint64(lowBytes, high)
	binary.LittleEndian.PutUint64(highBytes, low)

	guid := append(lowBytes, highBytes...)

	guidString := fmt.Sprintf("%x%x%x%x-%x%x-%x%x-%x-%x", guid[11], guid[10], guid[9], guid[8], guid[13], guid[12], guid[15], guid[14], guid[0:2], guid[2:8])
	return guidString
	// make one array shift first by 8
}

// ef22a537-a540-4988-871f-f85eeafd51d8
// 04ef22a5-3740-4988-871f-f85eeafd51d8
//a522ef0440378849871ff85eeafd51d8
