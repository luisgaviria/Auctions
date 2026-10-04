export interface Auction {
  id: number;
  address: string;
  city: string;
  state: string;
  time: string;
  logo: string;
  site_name: string;
  status: string;
  link: string;
  date: string;
  deposit: string;
  lat: string;
  lng: string;
  zillow_url: string;
  street_view_url: string;
  registry_url: string;
  registry_deep_link?: string;
  assessor_pid?: string;
  registry_book?: number;
  registry_page?: number;
}

export type AuctionCardData = Pick<Auction, 'id'> & Partial<Omit<Auction, 'id'>>;

export interface AuctionsResponse {
  message: string;
  auctions: Auction[];
}

export interface CityMarketSummary {
  city: string;
  county: string;
  auction_count: number;
  average_deposit: string;
}

export interface CityMarketResponse {
  summary: CityMarketSummary;
  auctions: Auction[];
}

export interface AuctionReport {
  auction: Auction;
  checklist: { task: string; category: string }[];
  share_url: string;
  address_slug: string;
  assessor_url?: string;
}

export interface CountyRow {
  county_slug: string;
  county: string;
  auction_count: number;
}

export interface CityRow {
  city_slug: string;
  city: string;
  county_slug: string;
  auction_count: number;
}

export interface CountiesResponse {
  counties: CountyRow[];
}

export interface CountyCitiesResponse {
  county: string;
  county_slug: string;
  cities: CityRow[];
}

export interface TopCitySlugsResponse {
  markets: CityRow[];
}

export interface AuctionResultsChangedDetail {
  auctions: Auction[];
}

export interface MarkerClickDetail {
  id: number;
  auction?: Auction;
}