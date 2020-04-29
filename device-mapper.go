package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	gosocketio "github.com/mtfelian/golang-socketio"
	fleet "github.com/synerex/proto_fleet"
	api "github.com/synerex/synerex_api"
	pbase "github.com/synerex/synerex_proto"
	sxutil "github.com/synerex/synerex_sxutil"
)

var (
	nodesrv         = flag.String("nodesrv", "127.0.0.1:9990", "Node ID Server")
	port            = flag.Int("port", 10080, "HarmoVis Provider Listening Port")
	mu              sync.Mutex
	version         = "0.01"
	assetsDir       http.FileSystem
	ioserv          *gosocketio.Server
	sxClient        *sxutil.SXServiceClient
	sxServerAddress string

	idMap map[int32]*DeviceID
)

const defaultIdMapFile = "idmap.json"

type DeviceID struct {
	ID             int32
	LastUpdateDate time.Time
}

func init() {
	log.Println("Starting UniqueID on Mapping Server..")
	rand.Seed(time.Now().UnixNano())

	if !loadIdMap() {
		idMap = make(map[int32]*DeviceID)
	}
}

func saveIdMap() {
	bytes, err := json.MarshalIndent(idMap, "", "  ")
	if err != nil {
		log.Printf("Cant marshal sxprofile")
	}
	err = ioutil.WriteFile(defaultIdMapFile, bytes, 0666)
	if err != nil {
		log.Println("Error on writing sxprofile.json ", err)
	}
}
func loadIdMap() bool {
	bytes, err := ioutil.ReadFile(defaultIdMapFile)
	if err != nil {
		log.Println("Error on reading sxprofile.json ", err)
		return false
	}
	jsonErr := json.Unmarshal(bytes, &idMap)
	if jsonErr != nil {
		log.Println("Can't unmarshall json ", jsonErr)
		return false
	}
	return true
}

// assetsFileHandler for static Data
func assetsFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return
	}

	file := r.URL.Path
	//	log.Printf("Open File '%s'",file)
	if file == "/" {
		file = "/index.html"
	}
	f, err := assetsDir.Open(file)
	if err != nil {
		log.Printf("can't open file %s: %v\n", file, err)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Printf("can't open file %s: %v\n", file, err)
		return
	}
	http.ServeContent(w, r, file, fi.ModTime(), f)
}

func run_server() *gosocketio.Server {

	currentRoot, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	d := filepath.Join(currentRoot, "mclient", "build")

	assetsDir = http.Dir(d)
	log.Println("AssetDir:", assetsDir)

	assetsDir = http.Dir(d)
	server := gosocketio.NewServer()

	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		log.Printf("Connected from %s as %s", c.IP(), c.Id())
	})

	server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		log.Printf("Disconnected from %s as %s", c.IP(), c.Id())
	})

	server.On("mapperID", func(c *gosocketio.Channel, _ string) {
		log.Printf("get reqest to make mapper ID from %s\n", c.Id())

		var rid int32
		for {
			rid = rand.Int31()
			_, ok := idMap[rid]
			if ok {
				continue
			}
			idMap[rid] = &DeviceID{
				ID:             rid,
				LastUpdateDate: time.Now(),
			}
			break
		}
		c.Emit("mapperID", strconv.Itoa(int(rid)))
		log.Printf("Create mapper ID: %d ", rid)
		saveIdMap()
	})

	server.On("latlon", func(c *gosocketio.Channel, latlon string) {
		log.Printf("latlon from %s at %s\n", c.Id(), latlon)
		// now need to send Synerex!
		sendFleet(c, latlon)
		//		return "OK"
	})

	return server
}

func monitorStatus() {
	for {
		sxutil.SetNodeStatus(int32(runtime.NumGoroutine()), "HV")
		time.Sleep(time.Second * 3)
	}
}

func sendFleet(c *gosocketio.Channel, latlon string) {

	//	this.state.socket.emit("latlon",""+this.mapperID+","+updateState.lat+","+updateState.lon+","+position.coords.heading+","+position.coords.speed+","+position.coords.alititude)

	var lat, lon, heading, speed, alt float64
	var mid int32
	n, _ := fmt.Sscanf(latlon, "%d,%f,%f,%f,%f,%f", &mid, &lat, &lon, &heading, &speed, &alt)

	_, ok := idMap[mid]
	if !ok {
		idMap[mid] = &DeviceID{
			ID:             mid,
			LastUpdateDate: time.Now(),
		}
	}

	fmt.Printf("%s: %d\n %d: %f,%f,%f,%f,%f", latlon, mid, n, lat, lon, heading, speed, alt)

	//

	fleet := fleet.Fleet{
		VehicleId: mid,
		Angle:     float32(heading),
		Speed:     int32(speed),
		Status:    int32(0),
		Coord: &fleet.Fleet_Coord{
			Lat: float32(lat),
			Lon: float32(lon),
		},
	}
	out, err := proto.Marshal(&fleet)
	if err == nil {
		cont := api.Content{Entity: out}
		// Register supply
		smo := sxutil.SupplyOpts{
			Name:  "Fleet Supply",
			Cdata: &cont,
		}
		_, nerr := sxClient.NotifySupply(&smo)
		if nerr != nil { // connection failuer with current client
			// we need to ask to nodeidserv?
			// or just reconnect.
			newClient := sxutil.GrpcConnectServer(sxServerAddress)
			if newClient != nil {
				log.Printf("Reconnect Server %s\n", sxServerAddress)
				sxClient.Client = newClient
			}
		}
	} else {
		log.Printf("PB Marshal Error!", err)
	}
}

func main() {
	log.Printf("Device-Mapper(%s) built %s sha1 %s", sxutil.GitVer, sxutil.BuildTime, sxutil.Sha1Ver)
	flag.Parse()

	channelTypes := []uint32{pbase.RIDE_SHARE}
	sxServerAddress, rerr := sxutil.RegisterNode(*nodesrv, "DeviceMapper", channelTypes, nil)
	if rerr != nil {
		log.Fatal("Can't register node ", rerr)
	}
	log.Printf("Connecting SynerexServer at [%s]\n", sxServerAddress)

	go sxutil.HandleSigInt()
	sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)

	wg := sync.WaitGroup{} // for syncing other goroutines

	ioserv = run_server()
	fmt.Printf("Running DeviceMapper Server..\n")
	if ioserv == nil {
		os.Exit(1)
	}

	client := sxutil.GrpcConnectServer(sxServerAddress) // if there is server address change, we should do it!

	argJson := fmt.Sprintf("{Client:Map:RIDE}")
	sxClient = sxutil.NewSXServiceClient(client, pbase.RIDE_SHARE, argJson)

	wg.Add(1)

	go monitorStatus() // keep status

	serveMux := http.NewServeMux()

	serveMux.Handle("/socket.io/", ioserv)
	serveMux.HandleFunc("/", assetsFileHandler)

	log.Printf("Starting Drvice Mapper Provider %s  on port %d", version, *port)
	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *port), serveMux)
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()

}
