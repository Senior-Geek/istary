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
	
	export class GoldMove {
	    names: string[];
	    before: number[];
	    after: number[];
	    total: number;
	    to: number;
	
	    static createFrom(source: any = {}) {
	        return new GoldMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.names = source["names"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.total = source["total"];
	        this.to = source["to"];
	    }
	}
	export class InnGuestView {
	    slot: number;
	    name: string;
	    refusal: string;
	
	    static createFrom(source: any = {}) {
	        return new InnGuestView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.name = source["name"];
	        this.refusal = source["refusal"];
	    }
	}
	export class InnQuote {
	    weeks: number;
	    cost: number;
	    hp: number;
	    healed: boolean;
	    needed: number;
	    affordable: number;
	    refusal: string;
	
	    static createFrom(source: any = {}) {
	        return new InnQuote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weeks = source["weeks"];
	        this.cost = source["cost"];
	        this.hp = source["hp"];
	        this.healed = source["healed"];
	        this.needed = source["needed"];
	        this.affordable = source["affordable"];
	        this.refusal = source["refusal"];
	    }
	}
	export class InnRoomView {
	    room: number;
	    key: string;
	    name: string;
	    hpPerWeek: number;
	    feePerWeek: number;
	    free: boolean;
	    untilHealed: InnQuote;
	    maxWeeks: number;
	
	    static createFrom(source: any = {}) {
	        return new InnRoomView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.room = source["room"];
	        this.key = source["key"];
	        this.name = source["name"];
	        this.hpPerWeek = source["hpPerWeek"];
	        this.feePerWeek = source["feePerWeek"];
	        this.free = source["free"];
	        this.untilHealed = this.convertValues(source["untilHealed"], InnQuote);
	        this.maxWeeks = source["maxWeeks"];
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
	export class InnStay {
	    guest: string;
	    room: number;
	    roomName: string;
	    weeks: number;
	    charged: number;
	    hpBefore: number;
	    hpAfter: number;
	    maxHpBefore: number;
	    maxHpAfter: number;
	    goldBefore: number;
	    goldAfter: number;
	    spells: boolean;
	    mageBefore: number[];
	    mageAfter: number[];
	    priestBefore: number[];
	    priestAfter: number[];
	    levelBefore: number;
	    levelAfter: number;
	    statsBefore: number[];
	    statsAfter: number[];
	    ageBefore: number;
	    ageAfter: number;
	    learned: string[];
	    status: string;
	    died: boolean;
	    messages: string[];
	    xpNeeded: number;
	
	    static createFrom(source: any = {}) {
	        return new InnStay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.guest = source["guest"];
	        this.room = source["room"];
	        this.roomName = source["roomName"];
	        this.weeks = source["weeks"];
	        this.charged = source["charged"];
	        this.hpBefore = source["hpBefore"];
	        this.hpAfter = source["hpAfter"];
	        this.maxHpBefore = source["maxHpBefore"];
	        this.maxHpAfter = source["maxHpAfter"];
	        this.goldBefore = source["goldBefore"];
	        this.goldAfter = source["goldAfter"];
	        this.spells = source["spells"];
	        this.mageBefore = source["mageBefore"];
	        this.mageAfter = source["mageAfter"];
	        this.priestBefore = source["priestBefore"];
	        this.priestAfter = source["priestAfter"];
	        this.levelBefore = source["levelBefore"];
	        this.levelAfter = source["levelAfter"];
	        this.statsBefore = source["statsBefore"];
	        this.statsAfter = source["statsAfter"];
	        this.ageBefore = source["ageBefore"];
	        this.ageAfter = source["ageAfter"];
	        this.learned = source["learned"];
	        this.status = source["status"];
	        this.died = source["died"];
	        this.messages = source["messages"];
	        this.xpNeeded = source["xpNeeded"];
	    }
	}
	export class InnView {
	    step: string;
	    guest: number;
	    name: string;
	    message: string;
	    messages: string[];
	    guests: InnGuestView[];
	    guestRefusal: string;
	    rooms: InnRoomView[];
	
	    static createFrom(source: any = {}) {
	        return new InnView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.guest = source["guest"];
	        this.name = source["name"];
	        this.message = source["message"];
	        this.messages = source["messages"];
	        this.guests = this.convertValues(source["guests"], InnGuestView);
	        this.guestRefusal = source["guestRefusal"];
	        this.rooms = this.convertValues(source["rooms"], InnRoomView);
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
	export class MapParty {
	    row: number;
	    col: number;
	    facing: string;
	    place: string;
	
	    static createFrom(source: any = {}) {
	        return new MapParty(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.col = source["col"];
	        this.facing = source["facing"];
	        this.place = source["place"];
	    }
	}
	export class MapFeature {
	    row: number;
	    col: number;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new MapFeature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.col = source["col"];
	        this.kind = source["kind"];
	    }
	}
	export class MapWall {
	    row: number;
	    col: number;
	    side: string;
	    kind: string;
	    oneWay?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MapWall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.col = source["col"];
	        this.side = source["side"];
	        this.kind = source["kind"];
	        this.oneWay = source["oneWay"];
	    }
	}
	export class MapSquare {
	    row: number;
	    col: number;
	
	    static createFrom(source: any = {}) {
	        return new MapSquare(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.col = source["col"];
	    }
	}
	export class LocalMapView {
	    level: number;
	    size: number;
	    visited: MapSquare[];
	    walls: MapWall[];
	    features: MapFeature[];
	    party: MapParty;
	    northEdge: number;
	    westEdge: number;
	
	    static createFrom(source: any = {}) {
	        return new LocalMapView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.size = source["size"];
	        this.visited = this.convertValues(source["visited"], MapSquare);
	        this.walls = this.convertValues(source["walls"], MapWall);
	        this.features = this.convertValues(source["features"], MapFeature);
	        this.party = this.convertValues(source["party"], MapParty);
	        this.northEdge = source["northEdge"];
	        this.westEdge = source["westEdge"];
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
	
	
	
	export class MapView {
	    level: number;
	    levels: number[];
	    size: number;
	    visited: MapSquare[];
	    walls: MapWall[];
	    wraps: MapWall[];
	    features: MapFeature[];
	    party?: MapParty;
	
	    static createFrom(source: any = {}) {
	        return new MapView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.levels = source["levels"];
	        this.size = source["size"];
	        this.visited = this.convertValues(source["visited"], MapSquare);
	        this.walls = this.convertValues(source["walls"], MapWall);
	        this.wraps = this.convertValues(source["wraps"], MapWall);
	        this.features = this.convertValues(source["features"], MapFeature);
	        this.party = this.convertValues(source["party"], MapParty);
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
	
	export class ShopOffer {
	    item: number;
	    name: string;
	    price: number;
	    stock: number;
	    usable: boolean;
	    refusal: string;
	
	    static createFrom(source: any = {}) {
	        return new ShopOffer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item = source["item"];
	        this.name = source["name"];
	        this.price = source["price"];
	        this.stock = source["stock"];
	        this.usable = source["usable"];
	        this.refusal = source["refusal"];
	    }
	}
	export class ShopOwned {
	    n: number;
	    item: number;
	    name: string;
	    identified: boolean;
	    equipped: boolean;
	    cursed: boolean;
	    price: number;
	    sellRefusal: string;
	    sellAsks: boolean;
	    fee: number;
	    uncurseRefusal: string;
	    identifyRefusal: string;
	
	    static createFrom(source: any = {}) {
	        return new ShopOwned(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.n = source["n"];
	        this.item = source["item"];
	        this.name = source["name"];
	        this.identified = source["identified"];
	        this.equipped = source["equipped"];
	        this.cursed = source["cursed"];
	        this.price = source["price"];
	        this.sellRefusal = source["sellRefusal"];
	        this.sellAsks = source["sellAsks"];
	        this.fee = source["fee"];
	        this.uncurseRefusal = source["uncurseRefusal"];
	        this.identifyRefusal = source["identifyRefusal"];
	    }
	}
	export class ShopView {
	    step: string;
	    prompt: boolean;
	    message: string;
	    customer: number;
	    name: string;
	    class: string;
	    gold: number;
	    confirm: number;
	    sellConfirm: number;
	    catalog: ShopOffer[];
	    items: ShopOwned[];
	    room: number;
	
	    static createFrom(source: any = {}) {
	        return new ShopView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.prompt = source["prompt"];
	        this.message = source["message"];
	        this.customer = source["customer"];
	        this.name = source["name"];
	        this.class = source["class"];
	        this.gold = source["gold"];
	        this.confirm = source["confirm"];
	        this.sellConfirm = source["sellConfirm"];
	        this.catalog = this.convertValues(source["catalog"], ShopOffer);
	        this.items = this.convertValues(source["items"], ShopOwned);
	        this.room = source["room"];
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
	export class TemplePatient {
	    index: number;
	    name: string;
	    class: string;
	    level: number;
	    status: string;
	    inParty: boolean;
	    kind: string;
	    donation: number;
	    refusal: string;
	    funding: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplePatient(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.class = source["class"];
	        this.level = source["level"];
	        this.status = source["status"];
	        this.inParty = source["inParty"];
	        this.kind = source["kind"];
	        this.donation = source["donation"];
	        this.refusal = source["refusal"];
	        this.funding = source["funding"];
	    }
	}
	export class TemplePayer {
	    slot: number;
	    name: string;
	    gold: number;
	    refusal: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplePayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.name = source["name"];
	        this.gold = source["gold"];
	        this.refusal = source["refusal"];
	    }
	}
	export class TempleShare {
	    slot: number;
	    name: string;
	    gold: number;
	
	    static createFrom(source: any = {}) {
	        return new TempleShare(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.name = source["name"];
	        this.gold = source["gold"];
	    }
	}
	export class TempleQuote {
	    patient: string;
	    donation: number;
	    payer: string;
	    payerGold: number;
	    partyGold: number;
	    mode: string;
	    shares: TempleShare[];
	    refusal: string;
	
	    static createFrom(source: any = {}) {
	        return new TempleQuote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.patient = source["patient"];
	        this.donation = source["donation"];
	        this.payer = source["payer"];
	        this.payerGold = source["payerGold"];
	        this.partyGold = source["partyGold"];
	        this.mode = source["mode"];
	        this.shares = this.convertValues(source["shares"], TempleShare);
	        this.refusal = source["refusal"];
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
	export class TempleReceipt {
	    patient: string;
	    index: number;
	    kind: string;
	    statusBefore: string;
	    statusAfter: string;
	    healed: boolean;
	    donation: number;
	    payer: string;
	    payerBefore: number;
	    payerAfter: number;
	    shares: TempleShare[];
	    party?: GoldMove;
	    hpBefore: number;
	    hpAfter: number;
	    maxHp: number;
	    ageBefore: number;
	    ageAfter: number;
	    messages: string[];
	
	    static createFrom(source: any = {}) {
	        return new TempleReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.patient = source["patient"];
	        this.index = source["index"];
	        this.kind = source["kind"];
	        this.statusBefore = source["statusBefore"];
	        this.statusAfter = source["statusAfter"];
	        this.healed = source["healed"];
	        this.donation = source["donation"];
	        this.payer = source["payer"];
	        this.payerBefore = source["payerBefore"];
	        this.payerAfter = source["payerAfter"];
	        this.shares = this.convertValues(source["shares"], TempleShare);
	        this.party = this.convertValues(source["party"], GoldMove);
	        this.hpBefore = source["hpBefore"];
	        this.hpAfter = source["hpAfter"];
	        this.maxHp = source["maxHp"];
	        this.ageBefore = source["ageBefore"];
	        this.ageAfter = source["ageAfter"];
	        this.messages = source["messages"];
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
	
	export class TempleView {
	    step: string;
	    message: string;
	    patients: TemplePatient[];
	    payers: TemplePayer[];
	    partyGold: number;
	
	    static createFrom(source: any = {}) {
	        return new TempleView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.message = source["message"];
	        this.patients = this.convertValues(source["patients"], TemplePatient);
	        this.payers = this.convertValues(source["payers"], TemplePayer);
	        this.partyGold = source["partyGold"];
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
	    n: number;
	    kind: string;
	    equipRefusal: string;
	    unequipRefusal: string;
	
	    static createFrom(source: any = {}) {
	        return new ItemView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.equipped = source["equipped"];
	        this.cursed = source["cursed"];
	        this.identified = source["identified"];
	        this.n = source["n"];
	        this.kind = source["kind"];
	        this.equipRefusal = source["equipRefusal"];
	        this.unequipRefusal = source["unequipRefusal"];
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
	    equipNote: string;
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
	        this.equipNote = source["equipNote"];
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
	export class EquipRequest {
	    index: number;
	    n: number;
	    equip: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EquipRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.n = source["n"];
	        this.equip = source["equip"];
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
	export class GoldReceipt {
	    move?: app.GoldMove;
	    saved: boolean;
	    durable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GoldReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.move = this.convertValues(source["move"], app.GoldMove);
	        this.saved = source["saved"];
	        this.durable = source["durable"];
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
	export class GoldRequest {
	    action: string;
	    slot: number;
	
	    static createFrom(source: any = {}) {
	        return new GoldRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.slot = source["slot"];
	    }
	}
	
	export class InnReceipt {
	    stay?: app.InnStay;
	    gold?: app.GoldMove;
	    saved: boolean;
	    durable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InnReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stay = this.convertValues(source["stay"], app.InnStay);
	        this.gold = this.convertValues(source["gold"], app.GoldMove);
	        this.saved = source["saved"];
	        this.durable = source["durable"];
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
	export class InnPanel {
	    view: app.InnView;
	    last?: InnReceipt;
	
	    static createFrom(source: any = {}) {
	        return new InnPanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.view = this.convertValues(source["view"], app.InnView);
	        this.last = this.convertValues(source["last"], InnReceipt);
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
	
	export class InnRequest {
	    action: string;
	    slot: number;
	    room: number;
	    weeks: number;
	
	    static createFrom(source: any = {}) {
	        return new InnRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.slot = source["slot"];
	        this.room = source["room"];
	        this.weeks = source["weeks"];
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
	    minimapVisible: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.partyVisible = source["partyVisible"];
	        this.minimapVisible = source["minimapVisible"];
	        this.note = source["note"];
	    }
	}
	export class ShopReceipt {
	    action: string;
	    customer: string;
	    item: string;
	    gold: number;
	    move?: app.GoldMove;
	    saved: boolean;
	    durable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShopReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.customer = source["customer"];
	        this.item = source["item"];
	        this.gold = source["gold"];
	        this.move = this.convertValues(source["move"], app.GoldMove);
	        this.saved = source["saved"];
	        this.durable = source["durable"];
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
	export class ShopPanel {
	    view: app.ShopView;
	    last?: ShopReceipt;
	
	    static createFrom(source: any = {}) {
	        return new ShopPanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.view = this.convertValues(source["view"], app.ShopView);
	        this.last = this.convertValues(source["last"], ShopReceipt);
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
	
	export class ShopRequest {
	    action: string;
	    slot: number;
	    item: number;
	    n: number;
	    yes: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShopRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.slot = source["slot"];
	        this.item = source["item"];
	        this.n = source["n"];
	        this.yes = source["yes"];
	    }
	}
	export class TempleLast {
	    treat?: app.TempleReceipt;
	    gold?: app.GoldMove;
	    saved: boolean;
	    durable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TempleLast(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.treat = this.convertValues(source["treat"], app.TempleReceipt);
	        this.gold = this.convertValues(source["gold"], app.GoldMove);
	        this.saved = source["saved"];
	        this.durable = source["durable"];
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
	export class TemplePanel {
	    view: app.TempleView;
	    last?: TempleLast;
	
	    static createFrom(source: any = {}) {
	        return new TemplePanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.view = this.convertValues(source["view"], app.TempleView);
	        this.last = this.convertValues(source["last"], TempleLast);
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
	export class TempleRequest {
	    action: string;
	    patient: number;
	    payer: number;
	    party: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TempleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.patient = source["patient"];
	        this.payer = source["payer"];
	        this.party = source["party"];
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
	    minimap?: app.LocalMapView;
	    camp?: CampPanel;
	    expedition: boolean;
	    real: boolean;
	    shop?: ShopPanel;
	    inn?: InnPanel;
	    gold?: GoldReceipt;
	    temple?: TemplePanel;
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
	        this.minimap = this.convertValues(source["minimap"], app.LocalMapView);
	        this.camp = this.convertValues(source["camp"], CampPanel);
	        this.expedition = source["expedition"];
	        this.real = source["real"];
	        this.shop = this.convertValues(source["shop"], ShopPanel);
	        this.inn = this.convertValues(source["inn"], InnPanel);
	        this.gold = this.convertValues(source["gold"], GoldReceipt);
	        this.temple = this.convertValues(source["temple"], TemplePanel);
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

