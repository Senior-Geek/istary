export namespace app {
	
	export class CampSpellView {
	    name: string;
	    book: string;
	    level: number;
	    target: boolean;
	    castable: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new CampSpellView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.book = source["book"];
	        this.level = source["level"];
	        this.target = source["target"];
	        this.castable = source["castable"];
	        this.reason = source["reason"];
	    }
	}
	export class DumapicView {
	    facing: string;
	    east: number;
	    north: number;
	    level: number;
	
	    static createFrom(source: any = {}) {
	        return new DumapicView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.facing = source["facing"];
	        this.east = source["east"];
	        this.north = source["north"];
	        this.level = source["level"];
	    }
	}
	export class EquipChoiceView {
	    number: number;
	    item: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new EquipChoiceView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.item = source["item"];
	        this.name = source["name"];
	    }
	}
	export class CampView {
	    screen: string;
	    message: string;
	    member: number;
	    placed?: number[];
	    category?: string;
	    choices?: EquipChoiceView[];
	    tradeWith: number;
	    spellBook?: string;
	    spell?: string;
	    dumapic?: DumapicView;
	    spells?: CampSpellView[];
	    actions: string[];
	
	    static createFrom(source: any = {}) {
	        return new CampView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.screen = source["screen"];
	        this.message = source["message"];
	        this.member = source["member"];
	        this.placed = source["placed"];
	        this.category = source["category"];
	        this.choices = this.convertValues(source["choices"], EquipChoiceView);
	        this.tradeWith = source["tradeWith"];
	        this.spellBook = source["spellBook"];
	        this.spell = source["spell"];
	        this.dumapic = this.convertValues(source["dumapic"], DumapicView);
	        this.spells = this.convertValues(source["spells"], CampSpellView);
	        this.actions = source["actions"];
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
	export class ChestMemberView {
	    slot: number;
	    name: string;
	    canAct: boolean;
	    inspected: boolean;
	    canCastCalfo: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChestMemberView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.name = source["name"];
	        this.canAct = source["canAct"];
	        this.inspected = source["inspected"];
	        this.canCastCalfo = source["canCastCalfo"];
	    }
	}
	export class ChestReportView {
	    slot: number;
	    member: string;
	    how: string;
	    said?: string;
	    setOff?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChestReportView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.member = source["member"];
	        this.how = source["how"];
	        this.said = source["said"];
	        this.setOff = source["setOff"];
	    }
	}
	export class CombatOptionView {
	    action: string;
	    key: string;
	    available: boolean;
	    reason?: string;
	    note?: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatOptionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.key = source["key"];
	        this.available = source["available"];
	        this.reason = source["reason"];
	        this.note = source["note"];
	        this.target = source["target"];
	    }
	}
	export class CombatChestView {
	    present: boolean;
	    stage?: string;
	    actions?: CombatOptionView[];
	    members?: ChestMemberView[];
	    reports?: ChestReportView[];
	    complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CombatChestView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.present = source["present"];
	        this.stage = source["stage"];
	        this.actions = this.convertValues(source["actions"], CombatOptionView);
	        this.members = this.convertValues(source["members"], ChestMemberView);
	        this.reports = this.convertValues(source["reports"], ChestReportView);
	        this.complete = source["complete"];
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
	export class CombatRefView {
	    side: string;
	    slot?: number;
	    group?: number;
	    member?: number;
	    name?: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatRefView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.side = source["side"];
	        this.slot = source["slot"];
	        this.group = source["group"];
	        this.member = source["member"];
	        this.name = source["name"];
	    }
	}
	export class CombatEffectView {
	    who: CombatRefView;
	    hpChange?: number;
	    statusFrom: string;
	    statusTo: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatEffectView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.who = this.convertValues(source["who"], CombatRefView);
	        this.hpChange = source["hpChange"];
	        this.statusFrom = source["statusFrom"];
	        this.statusTo = source["statusTo"];
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
	export class CombatEventView {
	    kind: string;
	    actor: CombatRefView;
	    target: CombatRefView;
	    lines: string[];
	    part?: number;
	    silent?: boolean;
	    effects?: CombatEffectView[];
	
	    static createFrom(source: any = {}) {
	        return new CombatEventView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.actor = this.convertValues(source["actor"], CombatRefView);
	        this.target = this.convertValues(source["target"], CombatRefView);
	        this.lines = source["lines"];
	        this.part = source["part"];
	        this.silent = source["silent"];
	        this.effects = this.convertValues(source["effects"], CombatEffectView);
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
	export class CombatGroupView {
	    number: number;
	    count: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatGroupView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.count = source["count"];
	        this.name = source["name"];
	    }
	}
	export class CombatItemView {
	    index: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatItemView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	    }
	}
	export class CombatMemberView {
	    slot: number;
	    name: string;
	    class: string;
	    ac: number;
	    hp: number;
	    maxHp: number;
	    status: string;
	    poisoned: boolean;
	    canAct: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CombatMemberView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.name = source["name"];
	        this.class = source["class"];
	        this.ac = source["ac"];
	        this.hp = source["hp"];
	        this.maxHp = source["maxHp"];
	        this.status = source["status"];
	        this.poisoned = source["poisoned"];
	        this.canAct = source["canAct"];
	    }
	}
	
	export class CombatOrderView {
	    slot: number;
	    member: string;
	    action: string;
	    group?: number;
	    spell?: string;
	    ally?: string;
	    item?: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatOrderView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.member = source["member"];
	        this.action = source["action"];
	        this.group = source["group"];
	        this.spell = source["spell"];
	        this.ally = source["ally"];
	        this.item = source["item"];
	    }
	}
	
	export class CombatSpellView {
	    name: string;
	    level: number;
	    kind: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new CombatSpellView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.level = source["level"];
	        this.kind = source["kind"];
	        this.target = source["target"];
	    }
	}
	export class CombatView {
	    phase: string;
	    next: string;
	    round: number;
	    frozen: boolean;
	    party: CombatMemberView[];
	    groups: CombatGroupView[];
	    actor: number;
	    menu: CombatOptionView[];
	    spells?: CombatSpellView[];
	    items?: CombatItemView[];
	    orders: CombatOrderView[];
	    events: CombatEventView[];
	    current: number;
	    lines: string[];
	    boons?: string[];
	    surprise?: string;
	    chest: CombatChestView;
	    xp?: number;
	    gold?: number;
	
	    static createFrom(source: any = {}) {
	        return new CombatView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.next = source["next"];
	        this.round = source["round"];
	        this.frozen = source["frozen"];
	        this.party = this.convertValues(source["party"], CombatMemberView);
	        this.groups = this.convertValues(source["groups"], CombatGroupView);
	        this.actor = source["actor"];
	        this.menu = this.convertValues(source["menu"], CombatOptionView);
	        this.spells = this.convertValues(source["spells"], CombatSpellView);
	        this.items = this.convertValues(source["items"], CombatItemView);
	        this.orders = this.convertValues(source["orders"], CombatOrderView);
	        this.events = this.convertValues(source["events"], CombatEventView);
	        this.current = source["current"];
	        this.lines = source["lines"];
	        this.boons = source["boons"];
	        this.surprise = source["surprise"];
	        this.chest = this.convertValues(source["chest"], CombatChestView);
	        this.xp = source["xp"];
	        this.gold = source["gold"];
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
	export class CorridorView {
	    left: string;
	    right: string;
	    front: string;
	    leftAhead?: string;
	    rightAhead?: string;
	
	    static createFrom(source: any = {}) {
	        return new CorridorView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.left = source["left"];
	        this.right = source["right"];
	        this.front = source["front"];
	        this.leftAhead = source["leftAhead"];
	        this.rightAhead = source["rightAhead"];
	    }
	}
	
	export class DungeonMemberView {
	    slot: number;
	    name: string;
	    class: string;
	    ac: number;
	    hp: number;
	    maxHp: number;
	    status: string;
	    poisoned: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DungeonMemberView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.name = source["name"];
	        this.class = source["class"];
	        this.ac = source["ac"];
	        this.hp = source["hp"];
	        this.maxHp = source["maxHp"];
	        this.status = source["status"];
	        this.poisoned = source["poisoned"];
	    }
	}
	export class DungeonView {
	    phase: string;
	    party: DungeonMemberView[];
	    corridor: CorridorView[];
	    dark: boolean;
	    viewport: string[];
	    lines: string[];
	    more: boolean;
	    prompt: string;
	    buttons: string[];
	    found: string[];
	    spells: string[];
	    actions: string[];
	
	    static createFrom(source: any = {}) {
	        return new DungeonView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.party = this.convertValues(source["party"], DungeonMemberView);
	        this.corridor = this.convertValues(source["corridor"], CorridorView);
	        this.dark = source["dark"];
	        this.viewport = source["viewport"];
	        this.lines = source["lines"];
	        this.more = source["more"];
	        this.prompt = source["prompt"];
	        this.buttons = source["buttons"];
	        this.found = source["found"];
	        this.spells = source["spells"];
	        this.actions = source["actions"];
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
	export class Unavailable {
	    action: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Unavailable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.reason = source["reason"];
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
	export class CampPanel {
	    view: app.CampView;
	    party: CharacterView[];
	    unavailable?: Unavailable[];
	
	    static createFrom(source: any = {}) {
	        return new CampPanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.view = this.convertValues(source["view"], app.CampView);
	        this.party = this.convertValues(source["party"], CharacterView);
	        this.unavailable = this.convertValues(source["unavailable"], Unavailable);
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
	export class CampRequest {
	    action: string;
	    slot: number;
	    n: number;
	    gold: number;
	    spell: string;
	    priest: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CampRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.slot = source["slot"];
	        this.n = source["n"];
	        this.gold = source["gold"];
	        this.spell = source["spell"];
	        this.priest = source["priest"];
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
	export class DeparturesView {
	    real: string;
	    practice: string;
	
	    static createFrom(source: any = {}) {
	        return new DeparturesView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.real = source["real"];
	        this.practice = source["practice"];
	    }
	}
	export class ExitView {
	    dest: string;
	    open: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ExitView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dest = source["dest"];
	        this.open = source["open"];
	        this.reason = source["reason"];
	    }
	}
	
	
	export class OrderRequest {
	    slot: number;
	    action: string;
	    group: number;
	    spell: string;
	    ally: number;
	    item: number;
	
	    static createFrom(source: any = {}) {
	        return new OrderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.action = source["action"];
	        this.group = source["group"];
	        this.spell = source["spell"];
	        this.ally = source["ally"];
	        this.item = source["item"];
	    }
	}
	export class PartyBarSlot {
	    slot: number;
	    index: number;
	    name: string;
	    class: string;
	    level: number;
	    ac: number;
	    hp: number;
	    maxHp: number;
	    status: string;
	    poisoned: boolean;
	    front: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PartyBarSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.index = source["index"];
	        this.name = source["name"];
	        this.class = source["class"];
	        this.level = source["level"];
	        this.ac = source["ac"];
	        this.hp = source["hp"];
	        this.maxHp = source["maxHp"];
	        this.status = source["status"];
	        this.poisoned = source["poisoned"];
	        this.front = source["front"];
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
	export class ReturnView {
	    open: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new ReturnView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.open = source["open"];
	        this.note = source["note"];
	    }
	}
	export class SettingsView {
	    partyVisible: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.partyVisible = source["partyVisible"];
	        this.note = source["note"];
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
	    location: string;
	    exits: ExitView[];
	    partyBar: PartyBarSlot[];
	    return?: ReturnView;
	    settings: SettingsView;
	    departures: DeparturesView;
	    party: PartySlotView[];
	    partyAlignment: string;
	    partyMax: number;
	    frontRow: number;
	    pendingJoin: string;
	    combat?: app.CombatView;
	    practiceChest: boolean;
	    dungeon?: app.DungeonView;
	    camp?: CampPanel;
	    expedition: boolean;
	    real: boolean;
	    unsaved: string;
	    warning: string;
	    recovery: string;
	    canReturn: boolean;
	
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
	        this.location = source["location"];
	        this.exits = this.convertValues(source["exits"], ExitView);
	        this.partyBar = this.convertValues(source["partyBar"], PartyBarSlot);
	        this.return = this.convertValues(source["return"], ReturnView);
	        this.settings = this.convertValues(source["settings"], SettingsView);
	        this.departures = this.convertValues(source["departures"], DeparturesView);
	        this.party = this.convertValues(source["party"], PartySlotView);
	        this.partyAlignment = source["partyAlignment"];
	        this.partyMax = source["partyMax"];
	        this.frontRow = source["frontRow"];
	        this.pendingJoin = source["pendingJoin"];
	        this.combat = this.convertValues(source["combat"], app.CombatView);
	        this.practiceChest = source["practiceChest"];
	        this.dungeon = this.convertValues(source["dungeon"], app.DungeonView);
	        this.camp = this.convertValues(source["camp"], CampPanel);
	        this.expedition = source["expedition"];
	        this.real = source["real"];
	        this.unsaved = source["unsaved"];
	        this.warning = source["warning"];
	        this.recovery = source["recovery"];
	        this.canReturn = source["canReturn"];
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

