/*
Application: Population Builder
Description: An app for estimating the population of a set of small areas
Filename: popbuilder.js
Copyright: Oliver Hawkins, 2015-16
Requires: D3 (d3), Leaflet (L)
*/

(function (window, document, d3, L) {

"use strict";

// Set up global application object
var pb = {};
window.pb = pb;

/* Constructor for the BoundarySearch object, a utility for determining 
which boundaries intersect with the map's current view. The boundaries
themselves are LSOAs and Data Zones grouped by local authority districts. 
LSOAs are used in England and Wales, Data Zones in Scotland. BoundarySearch 
finds the districts in view by searching within the regions in view. */
pb.BoundarySearch = function(regions) {

	this.regions = regions;
	this.districtsInView = [];

	this.updateBoundaryData = function(mapBounds) {

		var districtsInView = [],
			regionCodes = Object.keys(this.regions),
			regionBounds,
			districts,
			districtCode,
			districtCodes, 
			districtBounds,
			bounds;

		for (var i = 0; i < regionCodes.length; i++) {

			regionBounds = this.regions[regionCodes[i]].bounds;

			if (mapBounds.intersects(regionBounds)) {

				districts = this.regions[regionCodes[i]].districts;	
				districtCodes = Object.keys(districts);

				for (var j = 0; j < districtCodes.length; j++) {

					districtCode = districtCodes[j];
					districtBounds = districts[districtCode].bounds;

					if (mapBounds.intersects(districtBounds)) {

						districtsInView.push(districtCode);
					}
				}
			}
		}

		this.districtsInView = districtsInView;
	};
};

/* Constructor for the MapView object, a singleton that manages the state 
of the Leaflet map. */
pb.MapView = function(map) {

	this.map = map;
	this.popInfo = L.control();
	this.overlayControl = L.control({position: 'bottomright'});

	this.addDistrictLayer = function(districtLayer) {

		districtLayer.addTo(this.map);
	};

	this.removeDistrictLayer = function(districtLayer) {
	
		this.map.removeLayer(districtLayer);	
	};

	// Settings for the population information control
	this.popInfo.onAdd = function(map) {

		this._div = L.DomUtil.create('div', 'popinfo');
		this.update(0);
		return this._div;
	};

	// Updates the population information control with the given population
	this.popInfo.update = function(population) {

		if (population > 0) {

			var population = pb.numberWithCommas(population);
			this._div.innerHTML = '<h4>Population</h4><p><span ' +
				'class="number">' + population + '</span></p>' + 
				'<p><span class="action" ' +
				'onclick="pb.mapController.getResults();">' + 
				'Get data</span></p>';

		} else {

			var population = pb.numberWithCommas(population);
			this._div.innerHTML = '<h4>Population</h4>' + 
				'<p><span class="number">0</span></p>';
		}
	};

	// Add to map at start
	this.popInfo.addTo(this.map);

	// Settings for the overlay control
	this.overlayControl.onAdd = function(map) {

		this._div = L.DomUtil.create('div', 'overlaycontrol');
		this.update('Auto', false);
		return this._div;
	};

	// Updates the overlay control with the given population
	this.overlayControl.update = function(overlayState, zoneCode) {

		var zoneCode = (zoneCode !== '') ? zoneCode : '&hellip;';

		this._div.innerHTML = '<h4>Boundaries</h4><p><span class="action" ' + 
			'onclick="pb.mapController.changeOverlaySetting();">' + 
			overlayState + '</span></p><h4>Area Code</h4><p>' + 
			'<span class="code">' + zoneCode + '</span></p>' + 
			'<span class="action" onclick="pb.mapController.deselectAll();">' +
			'Clear Map</span></p>';
	};

	this.addOverlayControl = function() {

		this.overlayControl.addTo(this.map);
	};

	this.removeOverlayControl = function() {

		this.overlayControl.removeFrom(this.map);
	};	
};

/* Constructor for the MapModel object, a singleton that manages the state 
of the MapView. The MapController updates the MapModel with changes 
arising from events, and the MapModel updates the MapView. */
pb.MapModel = function(mapView) {

	this.mapView = mapView;
	this.mapBounds = mapView.map.getBounds();
	this.minimumZoom = 12;
	this.zoomLevel = 5;
	this.districtsInView = [];
	this.districtsLoaded = {};
	this.districtsOnMap = {};
	this.selectedFeatures = {};
	this.selectedZones = {};
	this.selectedPopulation = 0;
	this.overlayZoomLevel = 9;
	this.overlayControlActive = false;
	this.overlayStates = ['Auto', 'On', 'Off'];
	this.currentOverlayState = 0;
	this.highlightedZone = null;
	this.highlightedZoneCode = '';

	/* Method called when the map moves to update the map state.
	The method is given the codes of the districts in the current
	map view, as determined by the boundarySearch object. */
	this.setDistrictsInView = function(districtsInView) {

		var distInView,
			distOnMap,
			districtsOnMap = Object.keys(this.districtsOnMap);

		// Keep a record of the districts in view
		this.districtsInView = districtsInView;

		// Find districts in view that are not on the map and add them
		for (var i = 0; i < districtsInView.length; i++) {

			distInView = districtsInView[i];

			if (districtsOnMap.indexOf(distInView) == -1) {

				this.addDistrictToMap(distInView);
			}
		}

		// Find districts on the map that are not in view and remove them
		for (var j = 0; j < districtsOnMap.length; j++) {

			distOnMap = districtsOnMap[j];

			if (districtsInView.indexOf(distOnMap) == -1) {

				this.removeDistrictFromMap(distOnMap);
			}
		}
	};

	// Handles adding layers to the map and tracking their state
	this.addDistrictToMap = function(districtCode) {

		var mapModel = this,
			mapView = this.mapView,
			districtLayer;

		// The callback function used to retrieve json data for district layers
		var downloadDistrict = function(error, json) {

			// Stop and log an error if the json does not return
			if (error) return console.warn(error);

			if (!mapModel.districtsOnMap.hasOwnProperty(districtCode)) {

				districtLayer = L.geoJson(json, {
					className: districtCode, 
					color: '#A000A0', 
					weight: 2, 
					fillColor: '#D080D0',
					fillOpacity: 0,
					onEachFeature: function(feature, layer) {

						feature.properties.selected = false;

						layer.on('click', function(e) {
							
							if (feature.properties.selected) {

								mapModel.deselectZone(feature, e.target);
							
							} else {

								mapModel.selectZone(feature, e.target);
							}
						});

						layer.on('mouseover', function(e) {

							mapModel.setHighlightedZone(feature, e.target);
						});

						layer.on('mousemove', function(e) {

							mapModel.setCurrentZone(feature, e.target);
						});

						layer.on('mouseout', function(e) {

							mapModel.clearCurrentZone();
						});

						layer.on('contextmenu', function(e) {

							mapModel.setHighlightedZone(feature, e.target);
						});
					}
				});
				
				mapModel.districtsLoaded[districtCode] = districtLayer;

				if (mapModel.districtsInView.indexOf(districtCode) != -1) {

					mapModel.districtsOnMap[districtCode] = districtLayer;
					mapView.addDistrictLayer(districtLayer);
				}
			}
		};

		// Only add the layer if it is not already on the map
		if (!this.districtsOnMap.hasOwnProperty(districtCode)) {

			// If the layer has already been loaded then add it
			if (this.districtsLoaded.hasOwnProperty(districtCode)) {

				districtLayer = this.districtsLoaded[districtCode];
				this.districtsOnMap[districtCode] = districtLayer;
				mapView.addDistrictLayer(districtLayer);

			// Otherwise load the layer then add it with a callback
			} else {

				var jsonPath = 'resources/popzones/' + districtCode + '.json';
				d3.json(jsonPath, downloadDistrict);
			}
		}
	};

	// Handles removing layers from the map and tracking their state
	this.removeDistrictFromMap = function(districtCode) {

		var districtsLoaded = this.districtLoaded,
			districtsOnMap = this.districtsOnMap,
			districtLayer;

		// Only remove the layer if it is already on the map
		if (districtsOnMap.hasOwnProperty(districtCode)) {

			districtLayer = districtsOnMap[districtCode];
			this.mapView.removeDistrictLayer(districtLayer);
			delete districtsOnMap[districtCode];
		}
	};

	// Handles the selection of zones
	this.selectZone = function(feature, layer) {

		feature.properties.selected = true;
		this.selectedFeatures[feature.properties.zone] = feature;
		this.selectedZones[feature.properties.zone] = layer;
		this.selectedPopulation += parseInt(feature.properties.population, 10);
		layer.setStyle({fillOpacity: 0.4});
		this.mapView.popInfo.update(this.selectedPopulation);
	};

	// Handles the deselection of zones
	this.deselectZone = function(feature, layer) {

		feature.properties.selected = false;
		delete this.selectedFeatures[feature.properties.zone];
		delete this.selectedZones[feature.properties.zone];
		this.selectedPopulation -= parseInt(feature.properties.population, 10);
		layer.setStyle({fillOpacity: 0.0});
		this.mapView.popInfo.update(this.selectedPopulation);
	};

	// Deselects all sones on the map
	this.deselectAllZones = function() {

		for (var zoneCode in this.selectedFeatures) {

			var feature = this.selectedFeatures[zoneCode];
			var layer = this.selectedZones[zoneCode];
			this.deselectZone(feature, layer)
		}
	};

	// Sets the overlay state control setting to active
	this.activateOverlayControl = function() {

		if (this.overlayControlActive == false) {

			this.overlayControlActive = true;
			this.mapView.addOverlayControl();
			this.setOverlayState(this.currentOverlayState);
		}
	};

	// Sets the overlay state control setting to inactive
	this.deactivateOverlayControl = function() {

		if (this.overlayControlActive == true) {

			this.overlayControlActive = false;
			this.mapView.removeOverlayControl();
		}
	};

	// Sets the overlay state. Must be a valid overlay state.
	this.setOverlayState = function(overlayState) {

		this.currentOverlayState = overlayState;
		var nextOverlayState = this.overlayStates[overlayState];
		
		this.mapView.overlayControl.update(
			nextOverlayState, this.highlightedZoneCode);
	};

	// Sets the displayed zone code.
	this.setZoneCode = function(zoneCode) {

		var overlayState = this.currentOverlayState;
		var nextOverlayState = this.overlayStates[overlayState];
		this.mapView.overlayControl.update(nextOverlayState, zoneCode);
	};

	// Sets the current zone 
	this.setCurrentZone = function(feature, layer) {

		this.setZoneCode(feature.properties.zone);

		if (this.highlightedZone === null) {

			this.setHighlightedZone(feature, layer);
		} 
	};

	// Resets the current zone
	this.clearCurrentZone = function() {

		if (this.highlightedZone !== null) {
			
			this.highlightedZone.setStyle({color: '#A000A0', weight: 2});
			this.highlightedZone = null;
			this.highlightedZoneCode = '';
		}
		
		this.setZoneCode('');
	};

	// Sets the highlighted zone
	this.setHighlightedZone = function(feature, layer) {	

		if (this.highlightedZone !== null) {

			if (this.highlightedZone === layer) {

				this.clearCurrentZone();
				return;
			
			} else {

				this.highlightedZone.setStyle({color: '#A000A0', weight: 2});
				this.clearCurrentZone();
				this.highlightedZone = null;
				this.highlightedZoneCode = '';
			}
		}

		layer.setStyle({color: '#FF0080', weight: 8});
		this.highlightedZone = layer;
		this.highlightedZoneCode = feature.properties.zone;
		this.setCurrentZone(feature, layer);
	};
};

/* Constructor for the MapController object, a singleton that handles user 
events and updates the MapModel accordingly. */
pb.MapController = function(mapModel) {

	this.mapModel = mapModel;

	// Event handler called every time the user moves or zooms the map
	this.updateMap = function(mapBounds, newZoomLevel) {
		
		pb.boundarySearch.updateBoundaryData(mapBounds);
		var districtsInView = pb.boundarySearch.districtsInView;

		// Check whether the overlay control should be active at this zoom
		if (newZoomLevel > this.mapModel.overlayZoomLevel && 
			districtsInView.length > 0) {

			// If so, activate the overlay control
			mapModel.activateOverlayControl();
		
			// And decide whether to show overlays for the districts in view
			switch (this.mapModel.currentOverlayState) {

				// Auto
				case 0: 

					if (newZoomLevel > this.mapModel.minimumZoom) {

						this.mapModel.setDistrictsInView(districtsInView);
					
					} else {

						this.clearMap();
					}

					break;

				// On
				case 1: 

					this.mapModel.setDistrictsInView(districtsInView);
					break;

				// Off
				case 2: 

					this.clearMap();
					break;
			}

		} else {

			// If not, clear the map and deactivate the overlay control
			this.clearMap();
			mapModel.deactivateOverlayControl();
		}

		this.mapModel.mapBounds = mapBounds;
		this.mapModel.zoomLevel = newZoomLevel;
	};

	// Clears overlays from map and the current zone code
	this.clearMap = function() {

		this.mapModel.setDistrictsInView([]);
		this.mapModel.clearCurrentZone();
	};

	// Switches to the next overlay state, called by the overlay control
	this.changeOverlaySetting = function() {

		var overlayState = this.mapModel.currentOverlayState + 1;

		if (overlayState > this.mapModel.overlayStates.length - 1) {

			overlayState = 0;
		} 

		this.mapModel.setOverlayState(overlayState);
		this.updateMap(this.mapModel.mapBounds, this.mapModel.zoomLevel);
	};

	// Clears the selected areas
	this.deselectAll = function() {

		this.mapModel.deselectAllZones();
		this.mapModel.clearCurrentZone();
	}

	// Sends the selected areas to the results page
	this.getResults = function() {

		var selectedZoneCodes = Object.keys(this.mapModel.selectedZones);
		var zoneCodeString = selectedZoneCodes.join(',');
		var postParameters = {zones: zoneCodeString};
		var resultsPage = 'results';
		pb.submitForm(resultsPage, postParameters);
	};
};

// Load the bounds data, initialise the boundarySearch, and launch the app
pb.launch = function () {

	var boundsDataFile = 'resources/app/bounds.json';

	d3.json(boundsDataFile, function(boundsData) {

		pb.boundarySearch = new pb.BoundarySearch(boundsData.regions);
		pb.run();
	});
};

// The function to start the application, once the boundary search is loaded 
pb.run = function() { 

	// Initialise the map
	var map = L.map('map').setView([51.4997766, -0.1251731], 14);

	// Initialise the MVC objects
	var mapView = new pb.MapView(map);
	var mapModel = new pb.MapModel(mapView);
	var mapController = new pb.MapController(mapModel);

	// Add an interface to the mapController to the global object for controls
	pb.mapController = mapController; 

	// Preload wider boundaries for the start view
	var southWest = L.latLng(51.47047724507885, -0.24839401245117185),
		northEast = L.latLng(51.52903776845088, -0.0018882751464843748),
		widerBounds = L.latLngBounds(southWest, northEast);

	mapController.updateMap(widerBounds, map.getZoom());

	// Handle map movement
	map.on('moveend', function(e) {

		mapController.updateMap(map.getBounds(), map.getZoom());
	});

	// Add the tile layer

	/* Mapbox tile layer
	var tileLayerPath = 'https://api.tiles.mapbox.com/v4/mapbox.streets-basic/{z}/{x}/{y}.png?access_token={accessToken}',
		tileLayerAttribution = 'Map data &copy; <a href="http://openstreetmap.org">OpenStreetMap</a> contributors, <a href="http://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>, Imagery © <a href="http://mapbox.com">Mapbox</a>',
		tileLayerMaxZoom = 18,
		tileLayerId = 'Mapbox tile layer id goes here',
		tileLayerAccessToken = 'Mapbox api key goes here';
	
	L.tileLayer(tileLayerPath, {
		attribution: tileLayerAttribution,
		maxZoom: tileLayerMaxZoom,
		id: tileLayerId,
		accessToken: tileLayerAccessToken
	}).addTo(map);
	*/

	/* OpenStreetMap tile layer */
	L.tileLayer('http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
		maxZoom: 19,
		attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
	}).addTo(map);
};

// Utility function: Number formatter
pb.numberWithCommas = function(num) {

	var parts = num.toString().split(".");
	parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
	return parts.join(".");
};

// Utility function: Submit a post request
pb.submitForm = function(path, params, method) {

	method = method || "post"; 
 
	var form = document.createElement("form");
	form.setAttribute("method", method);
	form.setAttribute("action", path);
 
	/* Move the submit function to another variable
	so that it doesn't get overwritten */
	form._submit_function_ = form.submit;
 
	for(var key in params) {
		
		if(params.hasOwnProperty(key)) {

			var hiddenField = document.createElement("input");
			hiddenField.setAttribute("type", "hidden");
			hiddenField.setAttribute("name", key);
			hiddenField.setAttribute("value", params[key]);
			form.appendChild(hiddenField);
		 }
	}
 
	document.body.appendChild(form);
	form._submit_function_();
};

}(window, document, d3, L));