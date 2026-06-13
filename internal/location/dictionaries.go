package location

// The dictionaries below are seeded from the high-frequency location strings
// observed in production ATS data. They are meant to grow by observation — add
// the names/cities that show up unresolved, not a full gazetteer up front.

// regionCountries groups ISO 3166-1 alpha-2 country codes under one canonical
// region code from enrich.RegionValues. Each country maps to exactly one region
// (the coarse facet a user filters on); countryToRegion is the inverted lookup.
// "eu" is used in the broad geographic sense of Europe (not only EU members).
var regionCountries = map[string][]string{
	"eu": {
		"de", "fr", "nl", "es", "se", "pl", "ie", "pt", "it", "be", "dk",
		"fi", "at", "cz", "ro", "gr", "hu", "bg", "hr", "sk", "si", "lt",
		"lv", "ee", "lu", "ch", "no", "ua", "is",
	},
	"uk":            {"gb"},
	"us":            {"us"},
	"north_america": {"ca"},
	"latam":         {"ar", "br", "mx", "cl", "co", "pe", "uy"},
	"apac":          {"sg", "jp", "au", "nz", "in", "hk", "tw", "kr", "cn", "my", "th", "ph", "vn", "id"},
	"mena":          {"ae", "sa", "il", "eg", "tr", "qa"},
	"africa":        {"za", "ng", "ke"},
	"ru":            {"ru"},
}

// countryToRegion is the inverted regionCountries: ISO code -> region code.
var countryToRegion = invertRegionCountries()

func invertRegionCountries() map[string]string {
	out := make(map[string]string)
	for region, codes := range regionCountries {
		for _, code := range codes {
			out[code] = region
		}
	}
	return out
}

// nameToCountry resolves lowercase country names, common ATS shorthands, and a
// few beacon cities to an ISO 3166-1 alpha-2 code. The region falls out of
// countryToRegion, so shorthands like "uk" yield both the country (gb) and its
// region (uk) without a separate entry.
var nameToCountry = map[string]string{
	"united states": "us", "united states of america": "us",
	"usa": "us", "us": "us", "u.s.": "us", "u.s.a.": "us",
	"united kingdom": "gb", "uk": "gb", "u.k.": "gb",
	"england": "gb", "britain": "gb", "great britain": "gb", "london": "gb",
	"germany": "de", "deutschland": "de", "berlin": "de", "munich": "de", "münchen": "de", "hamburg": "de",
	"france": "fr", "paris": "fr",
	"netherlands": "nl", "the netherlands": "nl", "amsterdam": "nl",
	"spain": "es", "madrid": "es", "barcelona": "es",
	"sweden": "se", "stockholm": "se",
	"poland": "pl", "warsaw": "pl",
	"ireland": "ie", "dublin": "ie",
	"portugal": "pt", "lisbon": "pt",
	"italy": "it", "milan": "it", "rome": "it",
	"belgium": "be", "brussels": "be",
	"denmark": "dk", "copenhagen": "dk",
	"finland": "fi", "helsinki": "fi",
	"austria": "at", "vienna": "at",
	"switzerland": "ch", "zurich": "ch",
	"norway": "no", "ukraine": "ua",
	"canada": "ca", "toronto": "ca", "vancouver": "ca", "montreal": "ca",
	"singapore": "sg",
	"australia": "au", "sydney": "au", "melbourne": "au",
	"new zealand": "nz",
	"japan":       "jp", "tokyo": "jp",
	"india": "in", "pune": "in", "bangalore": "in", "bengaluru": "in", "mumbai": "in", "hyderabad": "in",
	"argentina": "ar", "brazil": "br", "mexico": "mx",
	"israel": "il", "tel aviv": "il",
	"united arab emirates": "ae", "dubai": "ae",
	"south africa": "za",
}

// nameToRegion resolves macro-region names (and explicit open-anywhere markers)
// directly to a region code, for tokens that name an area rather than a country.
var nameToRegion = map[string]string{
	"europe": "eu", "eu": "eu",
	"emea": "emea", "eea": "eea",
	"apac": "apac", "asia": "apac", "asia pacific": "apac", "asia-pacific": "apac",
	"americas":      "americas",
	"north america": "north_america",
	"latam":         "latam", "latin america": "latam", "south america": "latam",
	"mena": "mena", "middle east": "mena",
	"africa":   "africa",
	"anywhere": "global", "worldwide": "global", "global": "global", "remote anywhere": "global",
}
