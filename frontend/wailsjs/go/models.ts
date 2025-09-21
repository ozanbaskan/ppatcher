export namespace main {
	
	export class Config {
	    backend: string;
	    executable: string;
	    colorPalette: string;
	    mode: string;
	    version: string;
	    description: string;
	    logo: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backend = source["backend"];
	        this.executable = source["executable"];
	        this.colorPalette = source["colorPalette"];
	        this.mode = source["mode"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.logo = source["logo"];
	        this.icon = source["icon"];
	    }
	}

}

