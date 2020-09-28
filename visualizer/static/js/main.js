document.addEventListener('DOMContentLoaded', function() {
    document.querySelector(".send").addEventListener("click", distribute);  
    document.querySelector(".clear").addEventListener("click", clear); 
    getReceivers();

    setInterval(pollLoad, 100);
    setInterval(pollGenerators, 100);
});


function distribute() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            console.log("Fireing load - success");
            console.log(this.responseText);
         }
    };

    var select = document.querySelector("#receiver");
    var currentOpt = select.options[select.selectedIndex]; 
    var endpoint = currentOpt.value;
    var url = `/api/distribute?n=1000&c=1&url=${endpoint}/`


    xhttp.open("GET", url, true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function pollLoad() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            reportLoad(this.responseText);
         }
    };
    xhttp.open("GET", "/api/index", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function getReceivers() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        console.log("got a response")
        console.log(this.responseText);
         if (this.readyState == 4 && this.status == 200) {
            loadReceivers(this.responseText);
         }
    };
    xhttp.open("GET", "/api/receivers", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function reportLoad(content){
    var loadIndex = JSON.parse(content);


    for (var instance in loadIndex) {
        if (loadIndex.hasOwnProperty(instance)) {
            updateInstance(loadIndex[instance]);
        }
    };

    document.querySelector("#sentrequests .count").innerHTML= calculateCount(loadIndex);


}

function loadReceivers(content){
    var receivers = JSON.parse(content);
    var select = document.querySelector("#receiver");
    console.log(receivers);

    receivers.forEach(receiver => {
        console.log(receiver);
        var opt = document.createElement("option");
        opt.value = receiver.endpoint;
        opt.text = `${receiver.env} (${receiver.endpoint})  `;
        select.appendChild(opt);
    });
}

function calculateCount(instances){
    var total = 0;

    for (var instance in instances) {
        if (instances.hasOwnProperty(instance)) {
            total += instances[instance].count;
        }
    };


    return total;
}


function updateInstance(instance){

    var id = "#instance-" + instance.id
    var ui = document.querySelector(id);

    if (ui != null) {
        document.querySelector(id + " .count").innerHTML = instance.count;  
    } else {
        var envType = instance.env.toLowerCase();

        var imagePath = "";

        switch (envType) {
            case "cloudrun":
                imagePath = "img/cloudrun.svg";
              break;
            case "computeengine":
                imagePath = "img/computeengine.svg";
              break;
            case "appengine":
                imagePath = "img/appengine.svg";
              break;
            default :
                imagePath = "img/local.svg";
        }

        var envCont = document.querySelector("." + envType);

        if (envCont == null) {
            envCont = document.createElement("div");
            envCont.classList.add("envtype"); 
            envCont.classList.add(envType);
            var labelp = document.createElement("p");
            labelp.innerHTML = instance.env;
            envCont.appendChild(labelp);


            document.querySelector(".load-info").appendChild(envCont); 
            
        }


        var instanceDiv = document.createElement("div");
        instanceDiv.id = "instance-" + instance.id;
        instanceDiv.classList.add(envType);
        instanceDiv.classList.add("instance");
        
        var countDiv = document.createElement("div");
        countDiv.innerHTML = instance.count;  
        countDiv.classList.add("count");

        var idDiv = document.createElement("div");
        idDiv.innerHTML = instance.id;  
        idDiv.classList.add("id");
        
        var imgEle = document.createElement("img");
        imgEle.src =  imagePath;
        
        instanceDiv.appendChild(imgEle);
        instanceDiv.appendChild(idDiv);
        instanceDiv.appendChild(countDiv);

        envCont.appendChild(instanceDiv);

    }

}

function pollGenerators() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            reportGenerators(this.responseText);
         }
    };
    xhttp.open("GET", "/api/nodes", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function reportGenerators(content){

    var loadIndex = JSON.parse(content);

    for (var instance in loadIndex) {
        if (loadIndex.hasOwnProperty(instance)) {
            updateLoadGenerator(loadIndex[instance]);
        }
    };
}

function updateLoadGenerator(node){

    var id = "#generator-" + node.id

    var ui = document.querySelector(id);

    if (ui != null) {
        if (node.active){
           ui.classList.add("active");  
        } else{
           ui.classList.remove("active");  
        }
        
    } else {

        var nodeDiv = document.createElement("div");
        nodeDiv.id = "generator-" + node.id;
        nodeDiv.classList.add("node")

        var idDiv = document.createElement("div");
        idDiv.innerHTML = node.id;  
        idDiv.classList.add("id");
        
        var imgEle = document.createElement("img");
        imgEle.src =  "img/computeengine.svg";
        
        nodeDiv.appendChild(imgEle);
        nodeDiv.appendChild(idDiv);
        document.querySelector(".load-generators").appendChild(nodeDiv);

    }

}

function clear() {
    console.log("Clear called")
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            console.log(this.responseText);
            document.querySelector(".load-generators").innerHTML = "";
            document.querySelector(".load-info").innerHTML = "";
         }
    };
    xhttp.open("GET", "/api/clear", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}