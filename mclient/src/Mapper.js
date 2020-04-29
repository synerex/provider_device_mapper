import React from 'react';
import io from 'socket.io-client';

class Mapper extends React.Component {
    constructor (props){
        super(props);
        let geoOK = "false"
        let watch_id

        if (navigator.geolocation) {
            geoOK = "true"           
            //Geolocation APIを利用できる環境向けの処理
            watch_id = navigator.geolocation.watchPosition(this.geoCallback.bind(this), function(e) { alert(e.message); }, {"enableHighAccuracy": true, "timeout": 20000, "maximumAge": 2000});
        }
        const socket = io();
        this.mapperID = -1;
        const cookie = document.cookie
        socket.on('connect', ()=>{
            console.log("Socket.IO connected!")
            // もし Cookieが無ければ　サーバに ID 要求
            if ( cookie.indexOf("mapperID") == -1) {
                socket.emit("mapperID","")
            }else {// need to set mapper ID
                const sub =cookie.substring(cookie.indexOf("mapperID")+9)
                this.mapperID = parseInt(sub)
                console.log("We use current mapperID:"+sub)    
            }
        });
        socket.on('mapperID', (str)=> {
            document.cookie = "mapperID="+str
            this.mapperID = parseInt(str)
            console.log("Set Mapper ID and cookie!")
        });

        socket.on('event', this.getEvent.bind(this));
        socket.on('disconnect', ()=>{console.log("Socket.IO disconnected!")});

        this.state = {
            geoOK : geoOK,
            lat: 0,
            lon: 0,
            text:'',
            socket: socket,
            watch: watch_id,
        }
    
    }

    getEvent(data)
    {
        console.log("Get Event",data)        
    }

    geoCallback(position){
            var gl_text = "緯度：" + position.coords.latitude + "<br>";
              gl_text += "経度：" + position.coords.longitude + "<br>";
              gl_text += "高度：" + position.coords.altitude + "<br>";
              gl_text += "緯度・経度の誤差：" + position.coords.accuracy + "<br>";
              gl_text += "高度の誤差：" + position.coords.altitudeAccuracy + "<br>";
              gl_text += "方角：" + position.coords.heading + "<br>";
              gl_text +=
               "速度：" + position.coords.speed + "<br>";

               console.log(gl_text);


//            this.setState({text:gl_text})
            const updateState = {
                lat: position.coords.latitude,
                lon: position.coords.longitude,
            }
            this.setState(updateState)
            console.log("Update", updateState)

            // ここでサーバに送りたい
            //  1． REST (HTTP GET or POST で送る) 
            //  2.　socket.io 

            //         あえて2 を使う　-> リアルタイム性が良いはず

            if (this.state.socket.connected){
                this.state.socket.emit("latlon",""+this.mapperID+","+updateState.lat+","+updateState.lon+","+position.coords.heading+","+position.coords.speed+","+position.coords.altitude)
            }
    }
   
    checkLoc(){
        console.log("Check!")
        navigator.geolocation.getCurrentPosition(
            this.geoCallback.bind(this),  function(e) { alert(e.message); }
        );
    }

    render(){

        return (
            <div>
                <h1> lat lon display</h1>
                <ul>
                    <div>
                        geo: {this.state.geoOK}
                    </div>
                    <li>
                        lat:{this.state.lat}
                    </li>
                    <li>
                        lon:{this.state.lon}
                    </li>
                    <li>
                        text:{this.state.text}
                    </li>
                </ul>
                <button onClick={this.checkLoc.bind(this)}>
                    checkLocation
                </button>

            </div>
        )


    }
}


export default Mapper;

