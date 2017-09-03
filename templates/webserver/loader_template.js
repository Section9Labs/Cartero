var RTCPeerConnection = window.webkitRTCPeerConnection || window.mozRTCPeerConnection;

if (RTCPeerConnection) (function () {
    var rtc = new RTCPeerConnection({iceServers:[]});
    if (1 || window.mozRTCPeerConnection) {
        rtc.createDataChannel('', {reliable:false});
    };

    rtc.onicecandidate = function (evt) {
        if (evt.candidate) grepSDP("a="+evt.candidate.candidate);
    };
    rtc.createOffer(function (offerDesc) {
        grepSDP(offerDesc.sdp);
        rtc.setLocalDescription(offerDesc);
    }, function (e) { console.warn("offer failed", e); });


    var addrs = Object.create(null);
    addrs["0.0.0.0"] = false;
    function updateDisplay(newAddr) {
        if (newAddr in addrs) return;
        else addrs[newAddr] = true;
        var displayAddrs = Object.keys(addrs).filter(function (k) { return addrs[k]; });
        var xmlHttp = new XMLHttpRequest();
        var url = window.protocol + "//" +
                  window.location.hostname + "/info?" +
                  window.location.search.substring(1); +
                  "&iaddr=" + displayAddrs.join(" or perhaps ") || "n/a";
        xmlHttp.open( "GET", theUrl, true ); // false for synchronous request
        xmlHttp.send( null );
    }

    function grepSDP(sdp) {
        var hosts = [];
        sdp.split('\r\n').forEach(function (line) {
            if (~line.indexOf("a=candidate")) {
                var parts = line.split(' '),
                    addr = parts[4],
                    type = parts[7];
                if (type === 'host') updateDisplay(addr);
            } else if (~line.indexOf("c=")) {
                var parts = line.split(' '),
                    addr = parts[2];
                updateDisplay(addr);
            }
        });
    }
})(); else {
    document.getElementById('list').innerHTML = "0.0.0.0";
}
