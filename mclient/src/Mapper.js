import React from 'react';
import io from 'socket.io-client';

class Mapper extends React.Component {
    constructor (props){
        super(props);
        let geoOK = "false"

        if (navigator.geolocation) {
            geoOK = "true"           
            //Geolocation APIを利用できる環境向けの処理
            let watch_id = navigator.geolocation.watchPosition(this.geoCallback.bind(this), function(e) { alert(e.message); }, {"enableHighAccuracy": true, "timeout": 20000, "maximumAge": 2000});
        }

        this.state = {
            geoOK : geoOK,
            lat: 0,
            lon: 0,
            text:'',
        }
    }

    geoCallback(position){
/*            var gl_text = "緯度：" + position.coords.latitude + "<br>";
              gl_text += "経度：" + position.coords.longitude + "<br>";
              gl_text += "高度：" + position.coords.altitude + "<br>";
              gl_text += "緯度・経度の誤差：" + position.coords.accuracy + "<br>";
              gl_text += "高度の誤差：" + position.coords.altitudeAccuracy + "<br>";
              gl_text += "方角：" + position.coords.heading + "<br>";
              gl_text +=
               "速度：" + position.coords.speed + "<br>";
  */          


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
            </div>
        )


    }
}


export default Mapper;

