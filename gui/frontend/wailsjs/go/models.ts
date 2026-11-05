export namespace main {
	
	export class BackupInfo {
	    file: string;
	    date: string;
	    characters: number;
	    names: string[];
	    readable: boolean;
	    problem: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.date = source["date"];
	        this.characters = source["characters"];
	        this.names = source["names"];
	        this.readable = source["readable"];
	        this.problem = source["problem"];
	    }
	}
	export class ItemView {
	    name: string;
	    equipped: boolean;
	    cursed: boolean;
	    identified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ItemView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.equipped = source["equipped"];
	        this.cursed = source["cursed"];
	        this.identified = source["identified"];
	    }
	}
	export class CharacterView {
	    index: number;
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
	    xp: number;
	    nextXp: number;
	    hasPassword: boolean;
	    mageSpells: number[];
	    maxMage: number[];
	    priestSpells: number[];
	    maxPriest: number[];
	    spells: string[];
	    items: ItemView[];
	    poison: number;
	    inMaze: boolean;
	    mazeLevel: number;
	    legacy: boolean;
	    inParty: boolean;
	    partySlot: number;
	    joinRefusal: string;
	
	    static createFrom(source: any = {}) {
	        return new CharacterView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
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
	        this.xp = source["xp"];
	        this.nextXp = source["nextXp"];
	        this.hasPassword = source["hasPassword"];
	        this.mageSpells = source["mageSpells"];
	        this.maxMage = source["maxMage"];
	        this.priestSpells = source["priestSpells"];
	        this.maxPriest = source["maxPriest"];
	        this.spells = source["spells"];
	        this.items = this.convertValues(source["items"], ItemView);
	        this.poison = source["poison"];
	        this.inMaze = source["inMaze"];
	        this.mazeLevel = source["mazeLevel"];
	        this.legacy = source["legacy"];
	        this.inParty = source["inParty"];
	        this.partySlot = source["partySlot"];
	        this.joinRefusal = source["joinRefusal"];
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
	export class GuideRow {
	    class: string;
	    minimums: number[];
	    shortfall: number[];
	    alignments: string;
	    status: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new GuideRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.minimums = source["minimums"];
	        this.shortfall = source["shortfall"];
	        this.alignments = source["alignments"];
	        this.status = source["status"];
	        this.note = source["note"];
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
	    guide: GuideRow[];
	
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
	        this.guide = this.convertValues(source["guide"], GuideRow);
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
	
	
	export class PartySlotView {
	    slot: number;
	    index: number;
	    name: string;
	    short: string;
	    hp: number;
	    maxHp: number;
	    ac: number;
	    status: string;
	    front: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PartySlotView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.index = source["index"];
	        this.name = source["name"];
	        this.short = source["short"];
	        this.hp = source["hp"];
	        this.maxHp = source["maxHp"];
	        this.ac = source["ac"];
	        this.status = source["status"];
	        this.front = source["front"];
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
	    party: PartySlotView[];
	    partyAlignment: string;
	    partyMax: number;
	    frontRow: number;
	    pendingJoin: string;
	
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
	        this.party = this.convertValues(source["party"], PartySlotView);
	        this.partyAlignment = source["partyAlignment"];
	        this.partyMax = source["partyMax"];
	        this.frontRow = source["frontRow"];
	        this.pendingJoin = source["pendingJoin"];
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

