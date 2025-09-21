export namespace main {
	
	export class Config {
	    backend: string;
	    executable: string;
	    colorPalette: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backend = source["backend"];
	        this.executable = source["executable"];
	        this.colorPalette = source["colorPalette"];
	    }
	}

}

