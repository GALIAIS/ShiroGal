export namespace main {
	
	export class GameDetailsView {
	    id: number;
	    title_jp: string;
	    title_cn?: string;
	    brand: string;
	    release_date: string;
	    synopsis: string;
	    cover_url: string;
	    preview_urls?: string;
	    tags: string;
	    updated_at: string;
	    download_link?: string;
	
	    static createFrom(source: any = {}) {
	        return new GameDetailsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title_jp = source["title_jp"];
	        this.title_cn = source["title_cn"];
	        this.brand = source["brand"];
	        this.release_date = source["release_date"];
	        this.synopsis = source["synopsis"];
	        this.cover_url = source["cover_url"];
	        this.preview_urls = source["preview_urls"];
	        this.tags = source["tags"];
	        this.updated_at = source["updated_at"];
	        this.download_link = source["download_link"];
	    }
	}
	export class GameView {
	    ID: number;
	    TitleJP: string;
	    TitleCN: string;
	    Brand: string;
	    ReleaseDate: string;
	    CoverURL: string;
	
	    static createFrom(source: any = {}) {
	        return new GameView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.TitleJP = source["TitleJP"];
	        this.TitleCN = source["TitleCN"];
	        this.Brand = source["Brand"];
	        this.ReleaseDate = source["ReleaseDate"];
	        this.CoverURL = source["CoverURL"];
	    }
	}

}

