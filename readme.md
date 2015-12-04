# popbuilder
Population Builder is a web-application that allows the user to build a population estimate for a set of small areas in Great Britain. The application shows a full-screen map of Great Britain. When the user zooms into the map the boundaries of small areas are shown and these areas can be selected to add their estimated population to a total for all selected areas. By clicking "Get data" the user can get to a results screen that shows the population pyramid of the selected population in comparison to the population of Great Britain as a whole. A demonstration version of [Population Builder][pb] is available online.

### Demography

The areas used in the application are Lower-layer Super Output Areas (LSOAs) in England and Wales and DataZones in Scotland. The population estimates in the current version are the small area population estimates for mid-2014, which are published by the Office for National Statistics and the National Records of Scotland under the Open Government License. See the results page of the application for links to the original data sources.

### Technology

The server side of the application is written in Go, while the client-side uses Leaflet.js and D3. By default the application uses map tiles from OpenStreetMap, but the application JavaScript file popbuiler.js also contains the code to use Mapbox as the tile server instead. The code for using Mapbox is commented out. To use it simply uncomment the code, add your Mapbox API key details where indicated, and then remove or comment out the default OpenStreetMap code. The population data is stored on the server in two SQLite databases.

### Development Status

This is the first full working version of the application. It is narrowly focussed on the central task of producing custom population estimates, but it could potentially be expanded in a number of different directions, to include new data and/or new features. Certain aspects of the applcation could also potentially be modularised as re-usable components. This version has therefore been shared here as a starting point from which to branch out.

### Installation
Install the package with `go get`.

```sh
go get github.com/olihawkins/popbuilder
```

Type `popbuilder` in the application directory to start the application on port 3000. Go to http://localhost:3000 in a web browser to use it.

### Tests
Use `go test` to run the tests.

### Documentation
See the [GoDoc][gd] for the full documentation.

   [pb]: <http://olihawkins.com/projects/popbuilder>
   [gd]: <https://godoc.org/github.com/olihawkins/popbuilder>
