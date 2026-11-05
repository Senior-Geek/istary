export namespace main {
	
	export class CharacterView {
	    name: string;
	    race: string;
	    alignment: string;
	    class: string;
	    short: string;
	    level: number;
	    hp: number;
	    maxHp: number;
	    ac: number;
	    status: string;
	    gold: number;
	    age: number;
	    stats: number[];
	
	    static createFrom(source: any = {}) {
	        return new CharacterView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.race = source["race"];
	        this.alignment = source["alignment"];
	        this.class = source["class"];
	        this.short = source["short"];
	        this.level = source["level"];
	        this.hp = source["hp"];
	        this.maxHp = source["maxHp"];
	        this.ac = source["ac"];
	        this.status = source["status"];
	        this.gold = source["gold"];
	        this.age = source["age"];
	        this.stats = source["stats"];
	    }
	}
	export class CreationView {
	    step: string;
	    name: string;
	    hasPassword: boolean;
	    race: string;
	    alignment: string;
	    class: string;
	    stats: number[];
	    baseStats: number[];
	    points: number;
	    classOk: boolean[];
	    canAdd: boolean[];
	    canRemove: boolean[];
	
	    static createFrom(source: any = {}) {
	        return new CreationView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.name = source["name"];
	        this.hasPassword = source["hasPassword"];
	        this.race = source["race"];
	        this.alignment = source["alignment"];
	        this.class = source["class"];
	        this.stats = source["stats"];
	        this.baseStats = source["baseStats"];
	        this.points = source["points"];
	        this.classOk = source["classOk"];
	        this.canAdd = source["canAdd"];
	        this.canRemove = source["canRemove"];
	    }
	}
	export class View {
	    saveDir: string;
	    fatal: string;
	    error: string;
	    message: string;
	    notice: string;
	    screen: string;
	    roster: CharacterView[];
	    creation?: CreationView;
	    races: string[];
	    alignments: string[];
	    classes: string[];
	    statNames: string[];
	    nameMax: number;
	
	    static createFrom(source: any = {}) {
	        return new View(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saveDir = source["saveDir"];
	        this.fatal = source["fatal"];
	        this.error = source["error"];
	        this.message = source["message"];
	        this.notice = source["notice"];
	        this.screen = source["screen"];
	        this.roster = this.convertValues(source["roster"], CharacterView);
	        this.creation = this.convertValues(source["creation"], CreationView);
	        this.races = source["races"];
	        this.alignments = source["alignments"];
	        this.classes = source["classes"];
	        this.statNames = source["statNames"];
	        this.nameMax = source["nameMax"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

