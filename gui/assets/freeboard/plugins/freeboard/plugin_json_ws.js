(function()
{
  // Easy accessor to allow relative WS paths. (Works no matter if called from remote, localhost, local-ip, etc...)
  var wsURL = function() {
    var protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';
    var url = protocol + location.host;

    //if(location.hostname === 'localhost') {
    //    url += '/' + location.pathname.split('/')[1]; // add context path
    //}
    
    url += location.pathname.replace(/gui\/freeboard.*$/, "api/ws")
    return url;
  }

	var jsonWebSocketDatasource = function(settings, updateCallback)
	{
		var self = this;
		var currentSettings = settings;
		var ws;

    var raiseReloadOverlay=function()
    {
      document.body.innerHTML = "<h1>Connection Error</h1><h2>Please reload the page.</h2>";
    }
		
		var onOpen=function()
		{
			console.info("WebSocket(%s) Opened",currentSettings.url);
		}
		
		var onClose=function()
		{
			console.info("WebSocket Closed");
      raiseReloadOverlay();
		}
		
		var onMessage=function(event)
		{
			var data=event.data;
			
			console.info("WebSocket received %s",data);
			
			var objdata=JSON.parse(data);
			
			if(typeof objdata == "object")
			{
				updateCallback(objdata);
			}
			else
			{
				updateCallback(data);
			}
			
		}
		
		function createWebSocket()
		{
			if(ws) ws.close();
			
			var url=wsURL()+currentSettings.url;
			ws=new WebSocket(url);
			
			ws.onopen=onOpen;
			ws.onclose=onClose;
			ws.onmessage=onMessage;
		}
		
		//createWebSocket();

		this.updateNow = function()
		{
			createWebSocket();
		}

		this.onDispose = function()
		{
			ws.close();
		}

		this.onSettingsChanged = function(newSettings)
		{
			currentSettings = newSettings;
			
			createWebSocket();
		}
	};

	freeboard.loadDatasourcePlugin({
		type_name  : "JSON WebSocket",
		display_name : "JSON WebSocket Push Datasource",
		description : "A push datasource based on browser built-in WebSocket implementation, <em>by Lorenzo Maiorfi - <a href=\"http://www.innovactive.it\">Innovactive Engineering s.r.l.</a></em> You can download a sample dashboard using this datasource <a href=\"https://dl.dropboxusercontent.com/u/7414592/dashboard_demo_1.json\">here</a>",
		settings   : [
			{
				name        : "url",
				display_name: "URL",
				type        : "text"
			}
		],
		newInstance: function(settings, newInstanceCallback, updateCallback)
		{
			newInstanceCallback( new jsonWebSocketDatasource(settings, updateCallback));
		}
	});
}());
