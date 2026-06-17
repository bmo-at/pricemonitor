package stations

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/google/uuid"
	"github.com/sethvargo/go-retry"
)

const BrandShell Brand = "shell"

type StationShell struct {
	url   string
	brand Brand
}

type fuelLocalNames map[string]string

func (f *fuelLocalNames) UnmarshalJSON(data []byte) error {
	result := make(map[string]string)

	withoutPrefix, found := strings.CutPrefix(string(data), "\"{countryCode, select, ")

	if !found {
		return errors.New("")
	}

	parseable, found := strings.CutSuffix(withoutPrefix, "}}\"")

	if !found {
		return errors.New("")
	}

	for _, entry := range strings.Split(parseable, "} ") {
		split := strings.Split(entry, " {")

		countryCode := split[0]
		localFuelName := split[1]

		result[countryCode] = localFuelName
	}

	*f = (fuelLocalNames)(result)

	return nil
}

//nolint:tagliatelle // we do not control this, its from the shell react website
type ShellDataPage struct {
	Component string `json:"component"`
	Props     struct {
		Config struct {
			AlwaysShowOpeningHours bool `json:"alwaysShowOpeningHours"`
			Amenities              struct {
				Enabled  []string `json:"enabled"`
				Priority []string `json:"priority"`
			} `json:"amenities"`
			AppStoresBanner struct {
				GoogleLink string `json:"google_link"`
				IosLink    string `json:"ios_link"`
			} `json:"app_stores_banner"`
			Directions struct {
				Options struct {
					AvoidFerries   bool `json:"avoid_ferries"`
					AvoidHighways  bool `json:"avoid_highways"`
					AvoidTolls     bool `json:"avoid_tolls"`
					CorridorRadius struct {
						Range struct {
							Max int `json:"max"`
							Min int `json:"min"`
						} `json:"range"`
						Steps int `json:"steps"`
						Value int `json:"value"`
					} `json:"corridor_radius"`
					DrivingDistances bool   `json:"driving_distances"`
					TravelMode       string `json:"travel_mode"`
				} `json:"options"`
				UnitSystem     string `json:"unitSystem"`
				WaypointsLimit int    `json:"waypointsLimit"`
			} `json:"directions"`
			Directory struct {
				Path []string `json:"path"`
			} `json:"directory"`
			ExcludeFromDirectory []string `json:"exclude_from_directory"`
			FooterLinks          struct {
				Links struct {
					Accessibility string `json:"accessibility"`
					Facebook      string `json:"facebook"`
					Instagram     string `json:"instagram"`
					LinkedIn      string `json:"linked_in"`
					NearMe        string `json:"near_me"`
					Privacy       string `json:"privacy"`
					SiteLocator   string `json:"site_locator"`
					Twitter       string `json:"twitter"`
					Youtube       string `json:"youtube"`
				} `json:"links"`
				SectionOrder []string `json:"section_order"`
				Sections     struct {
					MoreLocation []string `json:"more_location"`
					MoreShell    []string `json:"more_shell"`
					Social       []string `json:"social"`
				} `json:"sections"`
			} `json:"footer_links"`
			FuelPricing struct {
				Enabled []string `json:"enabled"`
			} `json:"fuel_pricing"`
			Fuels struct {
				Enabled  []string `json:"enabled"`
				Priority []string `json:"priority"`
			} `json:"fuels"`
			HideFuelTypeToggles bool `json:"hideFuelTypeToggles"`
			InfoWindow          struct {
				Sections struct {
					Amenities struct {
						Enabled            []string `json:"enabled"`
						MaxStandaloneIcons int      `json:"maxStandaloneIcons"`
					} `json:"amenities"`
					FuelPricing struct {
						Enabled []string `json:"enabled"`
					} `json:"fuel_pricing"`
					TruckServices struct {
						Enabled            []string `json:"enabled"`
						MaxStandaloneIcons int      `json:"maxStandaloneIcons"`
					} `json:"truck_services"`
				} `json:"sections"`
			} `json:"info_window"`
			IntlData struct {
				Formats struct {
					List struct {
						Style string `json:"style"`
						Type  string `json:"type"`
					} `json:"list"`
					Number struct {
						Distance struct {
							Unit string `json:"unit"`
						} `json:"distance"`
					} `json:"number"`
				} `json:"formats"`
				Locales  []string `json:"locales"`
				Messages struct {
					ActionPanel struct {
						BackToResults string `json:"back_to_results"`
					} `json:"action_panel"`
					Amenities struct {
						AcsLocker                       string `json:"acs_locker"`
						AcsSmartPoint                   string `json:"acs_smart_point"`
						AdblueCar                       string `json:"adblue_car"`
						AdbluePack                      string `json:"adblue_pack"`
						AdblueTruck                     string `json:"adblue_truck"`
						AirAndWater                     string `json:"air_and_water"`
						Airmiles                        string `json:"airmiles"`
						AirmilesCash                    string `json:"airmiles_cash"`
						AlcoholicBeveragesBeer          string `json:"alcoholic_beverages_beer"`
						AlcoholicBeveragesSpirits       string `json:"alcoholic_beverages_spirits"`
						AlcoholicBeveragesWine          string `json:"alcoholic_beverages_wine"`
						Allsmart                        string `json:"allsmart"`
						AmazonLocker                    string `json:"amazon_locker"`
						ApplePay                        string `json:"apple_pay"`
						As24                            string `json:"as24"`
						Atm                             string `json:"atm"`
						AtmIn                           string `json:"atm_in"`
						AtmOut                          string `json:"atm_out"`
						AustrianHighwaySticker          string `json:"austrian_highway_sticker"`
						AutoElectrician                 string `json:"auto_electrician"`
						AutomotiveLpg                   string `json:"automotive_lpg"`
						B2BShellCardOta                 string `json:"b2b_shell_card_ota"`
						B2BShellCardQr                  string `json:"b2b_shell_card_qr"`
						BabyChangeFacilities            string `json:"baby_change_facilities"`
						BakeryShop                      string `json:"bakery_shop"`
						BankNoteAcceptor                string `json:"bank_note_acceptor"`
						BankOffice                      string `json:"bank_office"`
						Bchef                           string `json:"bchef"`
						Betting                         string `json:"betting"`
						BillaUnterwegs                  string `json:"billa_unterwegs"`
						Bilo                            string `json:"bilo"`
						Bonuslink                       string `json:"bonuslink"`
						BonusLink                       string `json:"bonus_link"`
						BottledGas                      string `json:"bottled_gas"`
						BrailleSignage                  string `json:"braille_signage"`
						Brakes                          string `json:"brakes"`
						BrazilianCafe                   string `json:"brazilian_cafe"`
						Budgens                         string `json:"budgens"`
						BulkPropane                     string `json:"bulk_propane"`
						Burger                          string `json:"burger"`
						BuyNowPayLater                  string `json:"buy_now_pay_later"`
						ByBox                           string `json:"by_box"`
						CaaCarWashDiscount              string `json:"caa_car_wash_discount"`
						CaaFuelDiscount                 string `json:"caa_fuel_discount"`
						CaaInStoreDiscount              string `json:"caa_in_store_discount"`
						Cafe                            string `json:"cafe"`
						CarCare                         string `json:"car_care"`
						CarRental                       string `json:"car_rental"`
						Carwash                         string `json:"carwash"`
						CarWash                         string `json:"car_wash"`
						CarWashHydrajet                 string `json:"car_wash_hydrajet"`
						CarWashMpay                     string `json:"car_wash_mpay"`
						CarwashMPay                     string `json:"carwash_m_pay"`
						CarWashSoftCloth                string `json:"car_wash_soft_cloth"`
						CarwashSubscription             string `json:"carwash_subscription"`
						CarWashTouchless                string `json:"car_wash_touchless"`
						Cash                            string `json:"cash"`
						CashDepositMachine              string `json:"cash_deposit_machine"`
						Cctv                            string `json:"cctv"`
						Charging                        string `json:"charging"`
						ChildrenPlayArea                string `json:"children_play_area"`
						ChildsToilet                    string `json:"childs_toilet"`
						CityMarket                      string `json:"city_market"`
						ClubsmartCard                   string `json:"clubsmart_card"`
						ClubsmartShop                   string `json:"clubsmart_shop"`
						CocaColaFreestyle               string `json:"coca_cola_freestyle"`
						CoffeeBeanToCup                 string `json:"coffee_bean_to_cup"`
						CoffeeDrip                      string `json:"coffee_drip"`
						Consultancy                     string `json:"consultancy"`
						Conveyor                        string `json:"conveyor"`
						ConveyorAndJet                  string `json:"conveyor_and_jet"`
						CoOp                            string `json:"co_op"`
						CoralGiftCard                   string `json:"coral_gift_card"`
						CoralPass                       string `json:"coral_pass"`
						Cords                           string `json:"cords"`
						CostaExpress                    string `json:"costa_express"`
						CostaExpressIcedDrinksAndHotTea string `json:"costa_express_iced_drinks_and_hot_tea"`
						CreditCard                      string `json:"credit_card"`
						CreditCardAmericanExpress       string `json:"credit_card_american_express"`
						CreditCardDinersClub            string `json:"credit_card_diners_club"`
						CreditCardGeneral               string `json:"credit_card_general"`
						CreditCardMastercard            string `json:"credit_card_mastercard"`
						CreditCardVisa                  string `json:"credit_card_visa"`
						CrtCard                         string `json:"crt_card"`
						Cumulus                         string `json:"cumulus"`
						CustomerService                 string `json:"customer_service"`
						CzechHighwaySticker             string `json:"czech_highway_sticker"`
						D4UCarWash                      string `json:"d4u_car_wash"`
						D4UPoints                       string `json:"d4u_points"`
						Defibrillators                  string `json:"defibrillators"`
						Deli2Go                         string `json:"deli2go"`
						DeliByShell                     string `json:"deli_by_shell"`
						DeliCafe                        string `json:"deli_cafe"`
						Deliveroo                       string `json:"deliveroo"`
						Dhl                             string `json:"dhl"`
						Diagnostics                     string `json:"diagnostics"`
						Dillons                         string `json:"dillons"`
						DisabilityAssistance            string `json:"disability_assistance"`
						DisabilityParking               string `json:"disability_parking"`
						DisabledFacilities              string `json:"disabled_facilities"`
						DocksideFueling                 string `json:"dockside_fueling"`
						DoorDash                        string `json:"door_dash"`
						DrinksTracker                   string `json:"drinks_tracker"`
						DrinkVendingMachine             string `json:"drink_vending_machine"`
						Drivein                         string `json:"drivein"`
						DunkinDonuts                    string `json:"dunkin_donuts"`
						EcotaxService                   string `json:"ecotax_service"`
						EdepotPlus                      string `json:"edepot_plus"`
						EnergyMastercard                string `json:"energy_mastercard"`
						Epass                           string `json:"epass"`
						EuroshellCard                   string `json:"euroshell_card"`
						ExclusiveProgramCarWash         string `json:"exclusive_program_car_wash"`
						ExclusiveProgramFuel            string `json:"exclusive_program_fuel"`
						ExclusiveProgramInStore         string `json:"exclusive_program_in_store"`
						FastFood                        string `json:"fast_food"`
						Filter                          string `json:"filter"`
						FinanceOptions                  string `json:"finance_options"`
						FleetCardDkv                    string `json:"fleet_card_dkv"`
						FleetCardEni                    string `json:"fleet_card_eni"`
						FleetCardEsso                   string `json:"fleet_card_esso"`
						FleetCardGeneral                string `json:"fleet_card_general"`
						FleetCardLotos                  string `json:"fleet_card_lotos"`
						FleetCardMolPolska              string `json:"fleet_card_mol_polska"`
						FleetCardMoya                   string `json:"fleet_card_moya"`
						FleetCardUta                    string `json:"fleet_card_uta"`
						FoodOfferings                   string `json:"food_offerings"`
						Foodpanda                       string `json:"foodpanda"`
						Forecourt                       string `json:"forecourt"`
						FourWLubeBayShell               string `json:"four_w_lube_bay_shell"`
						FourWLubeBayThirdParty          string `json:"four_w_lube_bay_third_party"`
						FredMeyer                       string `json:"fred_meyer"`
						FredMeyerMp                     string `json:"fred_meyer_mp"`
						FredMeyerQfc                    string `json:"fred_meyer_qfc"`
						FreeToilet                      string `json:"free_toilet"`
						Freshii                         string `json:"freshii"`
						FreshiiGrabAndGo                string `json:"freshii_grab_and_go"`
						Frys                            string `json:"frys"`
						FuelCard                        string `json:"fuel_card"`
						FuelDelivery                    string `json:"fuel_delivery"`
						FullService                     string `json:"full_service"`
						GermanPostalService             string `json:"german_postal_service"`
						GiantFoods                      string `json:"giant_foods"`
						GivingPump                      string `json:"giving_pump"`
						Gls                             string `json:"gls"`
						Go4More                         string `json:"go4more"`
						GooglePay                       string `json:"google_pay"`
						GoPlus                          string `json:"go_plus"`
						GuardedCarPark                  string `json:"guarded_car_park"`
						HairSalon                       string `json:"hair_salon"`
						HarrisTeeter                    string `json:"harris_teeter"`
						Harveys                         string `json:"harveys"`
						HdevOnly                        string `json:"hdev_only"`
						HeavyDutyEv                     string `json:"heavy_duty_ev"`
						HelixOilChange                  string `json:"helix_oil_change"`
						HelixServiceCentre              string `json:"helix_service_centre"`
						HgvLane                         string `json:"hgv_lane"`
						HighSpeedDieselPump             string `json:"high_speed_diesel_pump"`
						Hilander                        string `json:"hilander"`
						Homeland                        string `json:"homeland"`
						HotFood                         string `json:"hot_food"`
						HungarianHighwaySticker         string `json:"hungarian_highway_sticker"`
						Hybrid                          string `json:"hybrid"`
						Hyvee                           string `json:"hyvee"`
						IdoTicket                       string `json:"ido_ticket"`
						ILoveCafe                       string `json:"i_love_cafe"`
						InPost                          string `json:"in_post"`
						InStoreAssistance               string `json:"in_store_assistance"`
						InsuranceOffer                  string `json:"insurance_offer"`
						JamieOliverDeliByShell          string `json:"jamie_oliver_deli_by_shell"`
						JayC                            string `json:"jay_c"`
						Jet                             string `json:"jet"`
						JohnLewisClickAndCollect        string `json:"john_lewis_click_and_collect"`
						JustEat                         string `json:"just_eat"`
						KarcherCarWash                  string `json:"karcher_car_wash"`
						Kfc                             string `json:"kfc"`
						KingSoopers                     string `json:"king_soopers"`
						KingSoopersCityMarket           string `json:"king_soopers_city_market"`
						Kroger                          string `json:"kroger"`
						KrogerScotts                    string `json:"kroger_scotts"`
						Laundrette                      string `json:"laundrette"`
						LocalCommunityUse               string `json:"local_community_use"`
						Lottery                         string `json:"lottery"`
						LotteryTickets                  string `json:"lottery_tickets"`
						LoyaltyCards                    string `json:"loyalty_cards"`
						LoyaltyProgram                  string `json:"loyalty_program"`
						LubricantsDelivery              string `json:"lubricants_delivery"`
						LubricantsSales                 string `json:"lubricants_sales"`
						MaisonPradier                   string `json:"maison_pradier"`
						Manned                          string `json:"manned"`
						Manual                          string `json:"manual"`
						MautTerminal                    string `json:"maut_terminal"`
						Maxis                           string `json:"maxis"`
						Mcdonald                        string `json:"mcdonald"`
						Mcx                             string `json:"mcx"`
						Migrolino                       string `json:"migrolino"`
						MioShop                         string `json:"mio_shop"`
						MobileAirtime                   string `json:"mobile_airtime"`
						MobileLoyalty                   string `json:"mobile_loyalty"`
						MobilePayment                   string `json:"mobile_payment"`
						MobilePaymentAmex               string `json:"mobile_payment_amex"`
						MobilePaymentCartesBancaires    string `json:"mobile_payment_cartes_bancaires"`
						MobilePaymentDiners             string `json:"mobile_payment_diners"`
						MobilePaymentMaestro            string `json:"mobile_payment_maestro"`
						MobilePaymentMastercard         string `json:"mobile_payment_mastercard"`
						MobilePaymentVisa               string `json:"mobile_payment_visa"`
						MobilePaymentVisaClickToPay     string `json:"mobile_payment_visa_click_to_pay"`
						MondailRelayLocker              string `json:"mondail_relay_locker"`
						MondialRelayLocker              string `json:"mondial_relay_locker"`
						MoneyTransferServices           string `json:"money_transfer_services"`
						MotoCareExpress                 string `json:"moto_care_express"`
						MotorwayService                 string `json:"motorway_service"`
						Multinet                        string `json:"multinet"`
						NationalCard                    string `json:"national_card"`
						Navigator                       string `json:"navigator"`
						Navigator3CDiscount             string `json:"navigator_3c_discount"`
						Navigator5CDiscount             string `json:"navigator_5c_discount"`
						NavigatorFleetLoyaltyDiscount   string `json:"navigator_fleet_loyalty_discount"`
						Netpincer                       string `json:"netpincer"`
						Nitrogen                        string `json:"nitrogen"`
						NoOfShopSeats                   string `json:"no_of_shop_seats"`
						OilAndLubricants                string `json:"oil_and_lubricants"`
						OilRecycling                    string `json:"oil_recycling"`
						OnSiteFuelling                  string `json:"on_site_fuelling"`
						Opt                             string `json:"opt"`
						OtherLoyaltyCards               string `json:"other_loyalty_cards"`
						OtherOilChange                  string `json:"other_oil_change"`
						Others                          string `json:"others"`
						OtherThirdPartyRental           string `json:"other_third_party_rental"`
						Owens                           string `json:"owens"`
						P2CSupport                      string `json:"p2c_support"`
						PaidToilet                      string `json:"paid_toilet"`
						PaintShop                       string `json:"paint_shop"`
						ParkingLanes                    string `json:"parking_lanes"`
						PartnerCard                     string `json:"partner_card"`
						PartnerFleetCard                string `json:"partner_fleet_card"`
						PartnerLoyaltyAccepted          string `json:"partner_loyalty_accepted"`
						PayAtPump                       string `json:"pay_at_pump"`
						Paycell                         string `json:"paycell"`
						PayLess                         string `json:"pay_less"`
						PaymentKiosk                    string `json:"payment_kiosk"`
						PaymentType                     string `json:"payment_type"`
						Paypal                          string `json:"paypal"`
						PayTm                           string `json:"pay_tm"`
						PetFriendly                     string `json:"pet_friendly"`
						PetGrooming                     string `json:"pet_grooming"`
						Pharmacy                        string `json:"pharmacy"`
						PhoneShop                       string `json:"phone_shop"`
						Pizza                           string `json:"pizza"`
						PizzaHut                        string `json:"pizza_hut"`
						PizzaHutExpress                 string `json:"pizza_hut_express"`
						Plinto                          string `json:"plinto"`
						PostNl                          string `json:"post_nl"`
						PrayerRoom                      string `json:"prayer_room"`
						PrecisionTuneAutoCare           string `json:"precision_tune_auto_care"`
						PriceGuarantee                  string `json:"price_guarantee"`
						PropaneExchange                 string `json:"propane_exchange"`
						Qfc                             string `json:"qfc"`
						QuickLubes                      string `json:"quick_lubes"`
						Ralphs                          string `json:"ralphs"`
						RampAvailability                string `json:"ramp_availability"`
						Recup                           string `json:"recup"`
						Restaurant                      string `json:"restaurant"`
						Returns                         string `json:"returns"`
						RobertaCaffe                    string `json:"roberta_caffe"`
						Rollover                        string `json:"rollover"`
						RolloverAndJet                  string `json:"rollover_and_jet"`
						Roundys                         string `json:"roundys"`
						Sandwich                        string `json:"sandwich"`
						SaveMart                        string `json:"save_mart"`
						ScenePlus                       string `json:"scene_plus"`
						ScontiBancoposta                string `json:"sconti_bancoposta"`
						ScorecardPremiumPayback         string `json:"scorecard_premium_payback"`
						Scotts                          string `json:"scotts"`
						SecurityOffice                  string `json:"security_office"`
						Selectshop                      string `json:"selectshop"`
						SelfService                     string `json:"self_service"`
						SemiPublic                      string `json:"semi_public"`
						ServiceBay                      string `json:"service_bay"`
						ShellAdvance                    string `json:"shell_advance"`
						ShellAgro                       string `json:"shell_agro"`
						ShellCafe                       string `json:"shell_cafe"`
						ShellCard                       string `json:"shell_card"`
						ShellCard5CDiscount             string `json:"shell_card_5c_discount"`
						ShellCard7CDiscount             string `json:"shell_card_7c_discount"`
						ShellClubSmartExtraCard         string `json:"shell_club_smart_extra_card"`
						ShellDisabledAccess             string `json:"shell_disabled_access"`
						ShellDriversClub                string `json:"shell_drivers_club"`
						ShellEnergyRebate               string `json:"shell_energy_rebate"`
						ShellMachineRollover            string `json:"shell_machine_rollover"`
						Shop                            string `json:"shop"`
						Shower                          string `json:"shower"`
						SingleNetworkCard               string `json:"single_network_card"`
						SkipTheDishes                   string `json:"skip_the_dishes"`
						SkroutzLocker                   string `json:"skroutz_locker"`
						SlovakHighwaySticker            string `json:"slovak_highway_sticker"`
						SmartDeal                       string `json:"smart_deal"`
						SmartPay                        string `json:"smart_pay"`
						SmartShop                       string `json:"smart_shop"`
						SmartShopPremium                string `json:"smart_shop_premium"`
						Smiths                          string `json:"smiths"`
						SmithsFreddys                   string `json:"smiths_freddys"`
						SmithsFredMeyers                string `json:"smiths_fred_meyers"`
						SmogCheckServices               string `json:"smog_check_services"`
						SmokinBean                      string `json:"smokin_bean"`
						SnackFood                       string `json:"snack_food"`
						SnackVendingMachine             string `json:"snack_vending_machine"`
						SolarPanels                     string `json:"solar_panels"`
						SparExpress                     string `json:"spar_express"`
						SportsField                     string `json:"sports_field"`
						StandardToilet                  string `json:"standard_toilet"`
						Starbucks                       string `json:"starbucks"`
						SteerDiner                      string `json:"steer_diner"`
						Steers                          string `json:"steers"`
						StopAndShop                     string `json:"stop_and_shop"`
						SupermarketFuelDiscount         string `json:"supermarket_fuel_discount"`
						SwissHighwaySticker             string `json:"swiss_highway_sticker"`
						TbcCafe                         string `json:"tbc_cafe"`
						Tbd                             string `json:"tbd"`
						TelepassPremium                 string `json:"telepass_premium"`
						ThuisbezorgdNl                  string `json:"thuisbezorgd_nl"`
						Timewise                        string `json:"timewise"`
						TimHortons                      string `json:"tim_hortons"`
						Toilet                          string `json:"toilet"`
						TouchAndGo                      string `json:"touch_and_go"`
						TouchlessPay                    string `json:"touchless_pay"`
						TouchlessPayConvenience         string `json:"touchless_pay_convenience"`
						TouchlessPayFuels               string `json:"touchless_pay_fuels"`
						TrailerRental                   string `json:"trailer_rental"`
						TruckParking                    string `json:"truck_parking"`
						Truckport                       string `json:"truckport"`
						TruckWash                       string `json:"truck_wash"`
						TwentyFourHour                  string `json:"twenty_four_hour"`
						TwentyFourHourEvService         string `json:"twenty_four_hour_ev_service"`
						TwentyFourHourShop              string `json:"twenty_four_hour_shop"`
						Twotheloo                       string `json:"twotheloo"`
						TwoWLubeBay3RdParty             string `json:"two_w_lube_bay_3rd_party"`
						TwoWLubeBayShell                string `json:"two_w_lube_bay_shell"`
						TypeOfParking                   string `json:"type_of_parking"`
						TyreCentre                      string `json:"tyre_centre"`
						TyreService                     string `json:"tyre_service"`
						TyreWash                        string `json:"tyre_wash"`
						UberEats                        string `json:"uber_eats"`
						UberProOffer                    string `json:"uber_pro_offer"`
						Unmanned                        string `json:"unmanned"`
						UtilityPaymentServices          string `json:"utility_payment_services"`
						Utts                            string `json:"utts"`
						Vacuum                          string `json:"vacuum"`
						VehicleIdentitySystem           string `json:"vehicle_identity_system"`
						VehicleInspection               string `json:"vehicle_inspection"`
						VoorverpaktEnVers               string `json:"voorverpakt_en_vers"`
						Waitrose                        string `json:"waitrose"`
						WaterRefills                    string `json:"water_refills"`
						WheelchairAccessibleToilet      string `json:"wheelchair_accessible_toilet"`
						Wifi                            string `json:"wifi"`
						WindscreenRepair                string `json:"windscreen_repair"`
						WinnDixie                       string `json:"winn_dixie"`
						WirelessTerminals               string `json:"wireless_terminals"`
						Workspace                       string `json:"workspace"`
						Yellow                          string `json:"yellow"`
					} `json:"amenities"`
					AppStoresBanner struct {
						GoogleButtonAlt string `json:"google_button_alt"`
						IosButtonAlt    string `json:"ios_button_alt"`
						Title           string `json:"title"`
					} `json:"app_stores_banner"`
					Aria struct {
						CloseSection string `json:"close_section"`
						Header       struct {
							Menu string `json:"menu"`
						} `json:"header"`
						OpenSection string `json:"open_section"`
					} `json:"aria"`
					Autocomplete struct {
						Pending string `json:"pending"`
					} `json:"autocomplete"`
					Buttons struct {
						AddWaypoint    string `json:"add_waypoint"`
						Apply          string `json:"apply"`
						Cancel         string `json:"cancel"`
						Dismiss        string `json:"dismiss"`
						Minimize       string `json:"minimize"`
						Navigate       string `json:"navigate"`
						SendToPhone    string `json:"send_to_phone"`
						StationLocator string `json:"station_locator"`
						Unminimize     string `json:"unminimize"`
					} `json:"buttons"`
					CheckboxGroup struct {
						ShowLess string `json:"showLess"`
						ShowMore string `json:"showMore"`
					} `json:"checkbox_group"`
					ClusterTooltip struct {
						EvOnly     string `json:"ev_only"`
						FuelOnly   string `json:"fuel_only"`
						FuelPlusEv string `json:"fuel_plus_ev"`
					} `json:"cluster_tooltip"`
					Completions struct {
						Title string `json:"title"`
					} `json:"completions"`
					CountryCode struct {
						AE string `json:"AE"`
						AL string `json:"AL"`
						AR string `json:"AR"`
						AT string `json:"AT"`
						BA string `json:"BA"`
						BE string `json:"BE"`
						BG string `json:"BG"`
						CA string `json:"CA"`
						CH string `json:"CH"`
						CZ string `json:"CZ"`
						DE string `json:"DE"`
						DK string `json:"DK"`
						EE string `json:"EE"`
						FR string `json:"FR"`
						GB string `json:"GB"`
						HK string `json:"HK"`
						HR string `json:"HR"`
						HU string `json:"HU"`
						ID string `json:"ID"`
						IN string `json:"IN"`
						IT string `json:"IT"`
						LT string `json:"LT"`
						LU string `json:"LU"`
						LV string `json:"LV"`
						MK string `json:"MK"`
						MO string `json:"MO"`
						MX string `json:"MX"`
						MY string `json:"MY"`
						NL string `json:"NL"`
						OM string `json:"OM"`
						PH string `json:"PH"`
						PK string `json:"PK"`
						PL string `json:"PL"`
						PT string `json:"PT"`
						RS string `json:"RS"`
						RU string `json:"RU"`
						SG string `json:"SG"`
						SI string `json:"SI"`
						SK string `json:"SK"`
						TH string `json:"TH"`
						TR string `json:"TR"`
						UA string `json:"UA"`
						US string `json:"US"`
						XK string `json:"XK"`
						ZA string `json:"ZA"`
					} `json:"country_code"`
					DestinationHost struct {
						One63RetailPark     string `json:"163_retail_park"`
						AlbertHeijn         string `json:"albert_heijn"`
						Aldi                string `json:"aldi"`
						Annexum             string `json:"annexum"`
						Apc                 string `json:"apc"`
						Aprisco             string `json:"aprisco"`
						AsrRealEstate       string `json:"asr_real_estate"`
						Billa               string `json:"billa"`
						CarrefourMarket     string `json:"carrefour_market"`
						Gamma               string `json:"gamma"`
						Haje                string `json:"haje"`
						HypermarktCarrefour string `json:"hypermarkt_carrefour"`
						Intergamma          string `json:"intergamma"`
						IoiCityMall         string `json:"ioi_city_mall"`
						Jumbo               string `json:"jumbo"`
						Karwei              string `json:"karwei"`
						Kfc                 string `json:"kfc"`
						Lot10ShoppingCentre string `json:"lot_10_shopping_centre"`
						MarriottHotel       string `json:"marriott_hotel"`
						Penny               string `json:"penny"`
						Praxis              string `json:"praxis"`
						Reef                string `json:"reef"`
						Rewe                string `json:"rewe"`
						ShMintHotel         string `json:"sh_mint_hotel"`
						SunwayMedicalCentre string `json:"sunway_medical_centre"`
						SunwayPyramidMall   string `json:"sunway_pyramid_mall"`
						SunwayVelocity      string `json:"sunway_velocity"`
						TanglinMall         string `json:"tanglin_mall"`
						Tesco               string `json:"tesco"`
						TheMetropolis       string `json:"the_metropolis"`
						VanHerk             string `json:"van_herk"`
						Waitrose            string `json:"waitrose"`
					} `json:"destination_host"`
					Directory struct {
						List struct {
							Title struct {
								City      string `json:"city"`
								Locations string `json:"locations"`
								Root      string `json:"root"`
								State     string `json:"state"`
							} `json:"title"`
						} `json:"list"`
						RootBreadcrumb string `json:"root_breadcrumb"`
						RootSubtitle   string `json:"root_subtitle"`
						RootTitle      string `json:"root_title"`
						Subtitle       struct {
							City    string `json:"city"`
							Country string `json:"country"`
							State   string `json:"state"`
						} `json:"subtitle"`
						Title string `json:"title"`
					} `json:"directory"`
					Distance         string `json:"distance"`
					EvConnectorTypes struct {
						Ccs          string `json:"ccs"`
						Chademo      string `json:"chademo"`
						Domestic     string `json:"domestic"`
						GbtAc        string `json:"gbt_ac"`
						GbtDc        string `json:"gbt_dc"`
						Other        string `json:"other"`
						TepcoChademo string `json:"tepco_chademo"`
						Tesla        string `json:"tesla"`
						Type1        string `json:"type_1"`
						Type1Combo   string `json:"type_1_combo"`
						Type2        string `json:"type_2"`
						Type2Combo   string `json:"type_2_combo"`
						Type3        string `json:"type_3"`
					} `json:"ev_connector_types"`
					EvFeeTypes struct {
						AdHoc        string `json:"ad_hoc"`
						Blocking     string `json:"blocking"`
						Energy       string `json:"energy"`
						Session      string `json:"session"`
						Subscription string `json:"subscription"`
					} `json:"ev_fee_types"`
					EvPower struct {
						Fast      string `json:"fast"`
						HighPower string `json:"high_power"`
						Rapid     string `json:"rapid"`
						Slow      string `json:"slow"`
					} `json:"ev_power"`
					ExternalSites struct {
						EvLocations string `json:"ev_locations"`
					} `json:"external_sites"`
					FooterLinks struct {
						Links struct {
							Accessibility string `json:"accessibility"`
							Facebook      string `json:"facebook"`
							Instagram     string `json:"instagram"`
							LinkedIn      string `json:"linked_in"`
							NearMe        string `json:"near_me"`
							Privacy       string `json:"privacy"`
							SiteLocator   string `json:"site_locator"`
							Twitter       string `json:"twitter"`
							Youtube       string `json:"youtube"`
						} `json:"links"`
						Sections struct {
							MoreLocation string `json:"more_location"`
							MoreShell    string `json:"more_shell"`
							Social       string `json:"social"`
						} `json:"sections"`
					} `json:"footer_links"`
					Fuels struct {
						AutogasLpg               string `json:"autogas_lpg"`
						AutoRvPropane            string `json:"auto_rv_propane"`
						Biodiesel                string `json:"biodiesel"`
						BiofuelGasoline          string `json:"biofuel_gasoline"`
						ClearflexE85             string `json:"clearflex_e85"`
						Cng                      string `json:"cng"`
						DieselFit                string `json:"diesel_fit"`
						ElectricChargingOther    string `json:"electric_charging_other"`
						Fuelsave98               string `json:"fuelsave_98"`
						FuelsaveMidgradeGasoline string `json:"fuelsave_midgrade_gasoline"`
						FuelsaveRegularDiesel    string `json:"fuelsave_regular_diesel"`
						Gtl                      string `json:"gtl"`
						Hgo                      string `json:"hgo"`
						Hvo100HeatingOil         string `json:"hvo100_heating_oil"`
						Hydrogen                 string `json:"hydrogen"`
						Kerosene                 string `json:"kerosene"`
						Lng                      string `json:"lng"`
						LowOctaneGasoline        string `json:"low_octane_gasoline"`
						MidgradeGasoline         string `json:"midgrade_gasoline"`
						PremiumDiesel            string `json:"premium_diesel"`
						PremiumGasoline          string `json:"premium_gasoline"`
						RegularE15               string `json:"regular_e15"`
						Rng                      string `json:"rng"`
						ShellBiolng              string `json:"shell_biolng"`
						ShellHvo                 string `json:"shell_hvo"`
						ShellRecharge            string `json:"shell_recharge"`
						ShellRegularDiesel       string `json:"shell_regular_diesel"`
						ShellRenewableDiesel     string `json:"shell_renewable_diesel"`
						Super98                  string `json:"super98"`
						SuperPremiumGasoline     string `json:"super_premium_gasoline"`
						TruckDiesel              string `json:"truck_diesel"`
						UnleadedSuper            string `json:"unleaded_super"`
					} `json:"fuels"`
					FuelType struct {
						Conventional string `json:"conventional"`
						Ev           string `json:"ev"`
					} `json:"fuel_type"`
					Geolocation struct {
						Tooltip string `json:"tooltip"`
					} `json:"geolocation"`
					Info struct {
						StationLocator string `json:"station_locator"`
					} `json:"info"`
					InfoWindow struct {
						Aria struct {
							Image string `json:"image"`
						} `json:"aria"`
						BackToResults            string `json:"back_to_results"`
						Close                    string `json:"close"`
						DestinationHost          string `json:"destination_host"`
						Directions               string `json:"directions"`
						DirectionsLink           string `json:"directions_link"`
						DirectionsLinkNoDistance string `json:"directions_link_no_distance"`
						EvCharging               struct {
							ChargingPoints string `json:"charging_points"`
							Connectors     string `json:"connectors"`
							PaymentOptions struct {
								NewMotionApp string `json:"new_motion_app"`
								RfidToken    string `json:"rfid_token"`
								Title        string `json:"title"`
							} `json:"payment_options"`
							PowerAndUnit string `json:"power_and_unit"`
							Title        string `json:"title"`
						} `json:"ev_charging"`
						ForecourtHours        string `json:"forecourt_hours"`
						ForecourtOpeningHours struct {
							Title string `json:"title"`
						} `json:"forecourt_opening_hours"`
						FuelPrices struct {
							Available     string `json:"available"`
							Disclaimer    string `json:"disclaimer"`
							LastUpdatedOn string `json:"last_updated_on"`
							NotAvailable  string `json:"not_available"`
							NullPrice     string `json:"null_price"`
							Price         string `json:"price"`
							Timestamp     string `json:"timestamp"`
						} `json:"fuel_prices"`
						LocationID string `json:"location_id"`
						Mobile     struct {
							Telephone string `json:"telephone"`
						} `json:"mobile"`
						Nearby                  string `json:"nearby"`
						Offers                  string `json:"offers"`
						OnStreetChargerSubtitle string `json:"on_street_charger_subtitle"`
						OpeningHours            struct {
							Fri                 string `json:"fri"`
							Mon                 string `json:"mon"`
							OpenTwentyfourhours string `json:"open_twentyfourhours"`
							Period              string `json:"period"`
							Sat                 string `json:"sat"`
							Sun                 string `json:"sun"`
							Thu                 string `json:"thu"`
							Title               string `json:"title"`
							Tue                 string `json:"tue"`
							Wed                 string `json:"wed"`
						} `json:"opening_hours"`
						OpenStatus struct {
							OpenNow           string `json:"open_now"`
							OpenNowUntilLater string `json:"open_now_until_later"`
							OpenNowUntilToday string `json:"open_now_until_today"`
						} `json:"open_status"`
						Rebates  string `json:"rebates"`
						Sections struct {
							AdditionalInfo struct {
								Email     string `json:"email"`
								EvseID    string `json:"evse_id"`
								OcpiID    string `json:"ocpi_id"`
								Operator  string `json:"operator"`
								Telephone string `json:"telephone"`
								Title     string `json:"title"`
								Website   string `json:"website"`
							} `json:"additional_info"`
							Amenities struct {
								Title string `json:"title"`
							} `json:"amenities"`
							CarwashHours struct {
								Carwash string `json:"carwash"`
								Days    struct {
									Fri string `json:"Fri"`
									Mon string `json:"Mon"`
									Sat string `json:"Sat"`
									Sun string `json:"Sun"`
									Thu string `json:"Thu"`
									Tue string `json:"Tue"`
									Wed string `json:"Wed"`
								} `json:"days"`
								Hours       string `json:"hours"`
								NoHours     string `json:"no_hours"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"carwash_hours"`
							EvCharging struct {
								ConnectorAvailability string `json:"connector_availability"`
								ConnectorNameAndPower string `json:"connector_name_and_power"`
								Connectors            struct {
									Domestic     string `json:"domestic"`
									GbtAc        string `json:"gbt_ac"`
									GbtDc        string `json:"gbt_dc"`
									Other        string `json:"other"`
									TepcoChademo string `json:"tepco_chademo"`
									Tesla        string `json:"tesla"`
									Type1        string `json:"type_1"`
									Type1Combo   string `json:"type_1_combo"`
									Type2        string `json:"type_2"`
									Type2Combo   string `json:"type_2_combo"`
									Type3        string `json:"type_3"`
									Unspecified  string `json:"unspecified"`
								} `json:"connectors"`
								ConnectorTypes         string `json:"connector_types"`
								FurtherFees            string `json:"further_fees"`
								FurtherFeesTitle       string `json:"further_fees_title"`
								LastUpdated            string `json:"last_updated"`
								NumberOfChargingPoints string `json:"number_of_charging_points"`
								OperatorNames          string `json:"operator_names"`
								PointsOutOf            string `json:"points_out_of"`
								Power                  string `json:"power"`
								PowerAndCount          string `json:"power_and_count"`
								PricePerUnit           string `json:"price_per_unit"`
								SiteAvailability       string `json:"site_availability"`
								SiteCount              string `json:"site_count"`
								Title                  string `json:"title"`
							} `json:"ev_charging"`
							EvHours struct {
								Days struct {
									Fri string `json:"Fri"`
									Mon string `json:"Mon"`
									Sat string `json:"Sat"`
									Sun string `json:"Sun"`
									Thu string `json:"Thu"`
									Tue string `json:"Tue"`
									Wed string `json:"Wed"`
								} `json:"days"`
								Ev          string `json:"ev"`
								Hours       string `json:"hours"`
								NoHours     string `json:"no_hours"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"ev_hours"`
							ForecourtHours struct {
								Days struct {
									Fri string `json:"Fri"`
									Mon string `json:"Mon"`
									Sat string `json:"Sat"`
									Sun string `json:"Sun"`
									Thu string `json:"Thu"`
									Tue string `json:"Tue"`
									Wed string `json:"Wed"`
								} `json:"days"`
								Forecourt   string `json:"forecourt"`
								Hours       string `json:"hours"`
								NoHours     string `json:"no_hours"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"forecourt_hours"`
							FuelPricing struct {
								FuelPrices struct {
									Disclaimer string `json:"disclaimer"`
									NullPrice  string `json:"null_price"`
								} `json:"fuel_prices"`
								LastUpdated  string `json:"last_updated"`
								PricePerUnit string `json:"price_per_unit"`
								Title        string `json:"title"`
							} `json:"fuel_pricing"`
							Fuels struct {
								Title          string                    `json:"title"`
								FuelLocalNames map[string]fuelLocalNames `json:"fuel_local_names"`
							} `json:"fuels"`
							Hydrogen struct {
								PumpCount string `json:"pump_count"`
								PumpName  struct {
									H35 string `json:"h35"`
									H70 string `json:"h70"`
								} `json:"pump_name"`
								PumpTypes        string `json:"pump_types"`
								SupportedVehicle struct {
									Bus   string `json:"bus"`
									Car   string `json:"car"`
									Truck string `json:"truck"`
									Van   string `json:"van"`
								} `json:"supported_vehicle"`
								SupportedVehicles string `json:"supported_vehicles"`
								Title             string `json:"title"`
								Unavailable       string `json:"unavailable"`
							} `json:"hydrogen"`
							Info struct {
								Coordinates   string `json:"coordinates"`
								EsiCode       string `json:"esi_code"`
								ManningStatus struct {
									Manned   string `json:"manned"`
									Unmanned string `json:"unmanned"`
								} `json:"manning_status"`
								SiteID string `json:"site_id"`
							} `json:"info"`
							Links struct {
								Website string `json:"website"`
							} `json:"links"`
							Offers struct {
								Title string `json:"title"`
							} `json:"offers"`
							ShopHours struct {
								Days struct {
									Fri string `json:"Fri"`
									Mon string `json:"Mon"`
									Sat string `json:"Sat"`
									Sun string `json:"Sun"`
									Thu string `json:"Thu"`
									Tue string `json:"Tue"`
									Wed string `json:"Wed"`
								} `json:"days"`
								Hours       string `json:"hours"`
								NoHours     string `json:"no_hours"`
								Shop        string `json:"shop"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"shop_hours"`
							TruckServices struct {
								Title string `json:"title"`
							} `json:"truck_services"`
						} `json:"sections"`
						Services         string `json:"services"`
						ShopHours        string `json:"shop_hours"`
						ShopOpeningHours struct {
							Title string `json:"title"`
						} `json:"shop_opening_hours"`
						Title   string `json:"title"`
						Website string `json:"website"`
					} `json:"info_window"`
					Itinerary struct {
						Title string `json:"title"`
					} `json:"itinerary"`
					Links struct {
						Shell string `json:"shell"`
					} `json:"links"`
					Locations struct {
						ClosedTemporarily string `json:"closed_temporarily"`
						Description       string `json:"description"`
						Title             string `json:"title"`
					} `json:"locations"`
					LocationsList struct {
						Aria struct {
							Loading string `json:"loading"`
						} `json:"aria"`
						Buttons struct {
							Close string `json:"close"`
						} `json:"buttons"`
						Item struct {
							Telephone string `json:"telephone"`
						} `json:"item"`
					} `json:"locations_list"`
					LubeProducts struct {
						AeroShell                     string `json:"aero_shell"`
						AeroshellFluid                string `json:"aeroshell_fluid"`
						AeroshellGrease               string `json:"aeroshell_grease"`
						AeroshellPistonEngineOil      string `json:"aeroshell_piston_engine_oil"`
						AeroshellTurbineEngineOil     string `json:"aeroshell_turbine_engine_oil"`
						AirTool                       string `json:"air_tool"`
						Alexia                        string `json:"alexia"`
						Caprinus                      string `json:"caprinus"`
						Coolant                       string `json:"coolant"`
						Corena                        string `json:"corena"`
						Diala                         string `json:"diala"`
						Gadus                         string `json:"gadus"`
						Heat                          string `json:"heat"`
						HeatTransfer                  string `json:"heat_transfer"`
						Melina                        string `json:"melina"`
						Morlina                       string `json:"morlina"`
						Mysella                       string `json:"mysella"`
						Nautilus                      string `json:"nautilus"`
						Omala                         string `json:"omala"`
						Ondina                        string `json:"ondina"`
						Pennzoil                      string `json:"pennzoil"`
						PennzoilUltra                 string `json:"pennzoil_ultra"`
						POils                         string `json:"p_oils"`
						QuakerState                   string `json:"quaker_state"`
						Raizen                        string `json:"raizen"`
						Refrigeration                 string `json:"refrigeration"`
						RefrigerationOil              string `json:"refrigeration_oil"`
						Rotella                       string `json:"rotella"`
						ShellAdvance                  string `json:"shell_advance"`
						ShellArgina                   string `json:"shell_argina"`
						ShellFuelsavediesel           string `json:"shell_fuelsavediesel"`
						ShellGadinia                  string `json:"shell_gadinia"`
						ShellHeizoelEco               string `json:"shell_heizoel_eco"`
						ShellHeizoelEcoRenewableBlend string `json:"shell_heizoel_eco_renewable_blend"`
						ShellHeizL                    string `json:"shell_heizöl"`
						ShellHelix                    string `json:"shell_helix"`
						ShellIndustrial               string `json:"shell_industrial"`
						ShellRimula                   string `json:"shell_rimula"`
						ShellRotellaGasTruckMotorOil  string `json:"shell_rotella_gas_truck_motor_oil"`
						ShellSirius                   string `json:"shell_sirius"`
						ShellSpirax                   string `json:"shell_spirax"`
						ShellZone                     string `json:"shell_zone"`
						Spirax                        string `json:"spirax"`
						Tellus                        string `json:"tellus"`
						Textile                       string `json:"textile"`
						Tonna                         string `json:"tonna"`
						Turbo                         string `json:"turbo"`
						Urea                          string `json:"urea"`
					} `json:"lube_products"`
					Map struct {
						Legend struct {
							ButtonAria      string `json:"button_aria"`
							CloseButtonAria string `json:"close_button_aria"`
							Items           struct {
								ConventionalFuelSite       string `json:"conventional_fuel_site"`
								ConventionalFuelSiteWithEv string `json:"conventional_fuel_site_with_ev"`
								NfrOnlySite                string `json:"nfr_only_site"`
								OnStreet                   string `json:"on_street"`
								PartnerSiteNfrOnly         string `json:"partner_site_nfr_only"`
								PartnerSiteNoEv            string `json:"partner_site_no_ev"`
								PartnerSiteWithEv          string `json:"partner_site_with_ev"`
							} `json:"items"`
							Title string `json:"title"`
						} `json:"legend"`
					} `json:"map"`
					MinEvPower struct {
						Num3   string `json:"3"`
						Num5   string `json:"5"`
						Num7   string `json:"7"`
						Num22  string `json:"22"`
						Num43  string `json:"43"`
						Num50  string `json:"50"`
						Num70  string `json:"70"`
						Num150 string `json:"150"`
						Num175 string `json:"175"`
						Num350 string `json:"350"`
					} `json:"min_ev_power"`
					MobileNav struct {
						Tabs struct {
							List string `json:"list"`
							Map  string `json:"map"`
						} `json:"tabs"`
					} `json:"mobile_nav"`
					OpenNow struct {
						ClosedUntil string `json:"closed_until"`
						Open24Hours string `json:"open_24_hours"`
						OpenUntil   string `json:"open_until"`
					} `json:"open_now"`
					OpenStatuses struct {
						Forecourt string `json:"forecourt"`
						OpenNow   string `json:"open_now"`
						Shop      string `json:"shop"`
					} `json:"open_statuses"`
					OpenStatus struct {
						OpenNow           string `json:"open_now"`
						OpenNowUntilLater string `json:"open_now_until_later"`
						OpenNowUntilToday string `json:"open_now_until_today"`
					} `json:"open_status"`
					PaymentOptions struct {
						NewMotionApp string `json:"new_motion_app"`
						RfidToken    string `json:"rfid_token"`
					} `json:"payment_options"`
					RechargeAvailabilityStatus struct {
						Available string `json:"available"`
					} `json:"recharge_availability_status"`
					RouteOptions struct {
						Button struct {
							AriaCount string `json:"aria_count"`
							AriaText  string `json:"aria_text"`
						} `json:"button"`
						Close   string `json:"close"`
						Options struct {
							AvoidFerries  string `json:"avoid_ferries"`
							AvoidHighways string `json:"avoid_highways"`
							AvoidTolls    string `json:"avoid_tolls"`
						} `json:"options"`
						Radius struct {
							Heading string `json:"heading"`
							Label   string `json:"label"`
							Value   string `json:"value"`
						} `json:"radius"`
						Title string `json:"title"`
					} `json:"route_options"`
					ScrollWarning struct {
						CloseButton string `json:"close_button"`
						Text        string `json:"text"`
					} `json:"scroll_warning"`
					Search struct {
						Aria struct {
							BackButton       string `json:"back_button"`
							Clear            string `json:"clear"`
							DirectionsButton string `json:"directions_button"`
							Intro            string `json:"intro"`
							LocatorButton    string `json:"locator_button"`
							Results          string `json:"results"`
						} `json:"aria"`
						Completions struct {
							ClearAllRecent string `json:"clear_all_recent"`
							Recent         string `json:"recent"`
						} `json:"completions"`
						EnterLocation     string `json:"enter_location"`
						Geolocation       string `json:"geolocation"`
						GeolocationStatus struct {
							Error   string `json:"error"`
							None    string `json:"none"`
							Success string `json:"success"`
							Waiting string `json:"waiting"`
						} `json:"geolocation_status"`
						Help struct {
							Clear    string `json:"clear"`
							Inactive string `json:"inactive"`
						} `json:"help"`
						Placeholder    string `json:"placeholder"`
						RecentSearches string `json:"recent_searches"`
						Suggestions    string `json:"suggestions"`
						Title          string `json:"title"`
						UseMyLocation  string `json:"use_my_location"`
						YourLocation   string `json:"your_location"`
					} `json:"search"`
					Section struct {
						Feature string `json:"feature"`
					} `json:"section"`
					Sections struct {
						AnalyticsAcceptance struct {
							Allow  string `json:"allow"`
							Body   string `json:"body"`
							Refuse string `json:"refuse"`
							Title  string `json:"title"`
						} `json:"analytics_acceptance"`
						Footer struct {
							Links struct {
								Accessibility string `json:"accessibility"`
								Facebook      string `json:"facebook"`
								Instagram     string `json:"instagram"`
								LinkedIn      string `json:"linked_in"`
								NearMe        string `json:"near_me"`
								Privacy       string `json:"privacy"`
								SiteLocator   string `json:"site_locator"`
								Twitter       string `json:"twitter"`
								Youtube       string `json:"youtube"`
							} `json:"links"`
							Sections struct {
								MoreLocation string `json:"more_location"`
								MoreShell    string `json:"more_shell"`
								Social       string `json:"social"`
							} `json:"sections"`
						} `json:"footer"`
						Header struct {
							MobileTitle         string `json:"mobile_title"`
							SkipToContentButton string `json:"skip_to_content_button"`
						} `json:"header"`
						LocationsList struct {
							GeographicalItem string `json:"geographical_item"`
							NoResults        string `json:"no_results"`
							Search           struct {
								CityPlaceholder  string `json:"city_placeholder"`
								ClearLabel       string `json:"clear_label"`
								Label            string `json:"label"`
								Placeholder      string `json:"placeholder"`
								RootPlaceholder  string `json:"root_placeholder"`
								StatePlaceholder string `json:"state_placeholder"`
							} `json:"search"`
						} `json:"locations_list"`
						TopLocations struct {
							Item     string `json:"item"`
							Subtitle struct {
								Counties       string `json:"counties"`
								Other          string `json:"other"`
								States         string `json:"states"`
								TownsAndCities string `json:"towns_and_cities"`
							} `json:"subtitle"`
							Title struct {
								Root string `json:"root"`
							} `json:"title"`
						} `json:"top_locations"`
					} `json:"sections"`
					SendToPhone struct {
						CopyLink struct {
							Button string `json:"button"`
							Header string `json:"header"`
						} `json:"copy_link"`
						Mail struct {
							Body    string `json:"body"`
							Header  string `json:"header"`
							Link    string `json:"link"`
							Subject string `json:"subject"`
						} `json:"mail"`
						QrCodeHeader string `json:"qr_code_header"`
						Title        string `json:"title"`
					} `json:"send_to_phone"`
					ShopOpenStatuses struct {
						OpenNow string `json:"open_now"`
					} `json:"shop_open_statuses"`
					SiteStatus struct {
						ClosedTemporarily string `json:"closed_temporarily"`
					} `json:"site_status"`
					SiteType struct {
						HidePartnerSites string `json:"hide_partner_sites"`
					} `json:"site_type"`
					StationPage struct {
						Buttons struct {
							GetDirections  string `json:"get_directions"`
							Offers         string `json:"offers"`
							StationLocator string `json:"station_locator"`
						} `json:"buttons"`
						Description struct {
							EvCharging    string `json:"ev_charging"`
							FoodOfferings struct {
								Other    string `json:"other"`
								Plural   string `json:"plural"`
								Singular string `json:"singular"`
							} `json:"food_offerings"`
							FuelProducts struct {
								EvCharging struct {
									ConventionalFuelSiteWithEv string `json:"conventional_fuel_site_with_ev"`
									DestinationChargingEv      string `json:"destination_charging_ev"`
									HydrogenSite               string `json:"hydrogen_site"`
									LngSite                    string `json:"lng_site"`
									MobilityHubEvHub           string `json:"mobility_hub_ev_hub"`
									MobilityHubEvPlusHub       string `json:"mobility_hub_ev_plus_hub"`
									NewFuelHub                 string `json:"new_fuel_hub"`
								} `json:"ev_charging"`
								Hydrogen string `json:"hydrogen"`
								Products struct {
									NewFuelHub  string `json:"new_fuel_hub"`
									NfrOnlySite string `json:"nfr_only_site"`
									Other       string `json:"other"`
								} `json:"products"`
							} `json:"fuel_products"`
							Intro struct {
								Basic                      string `json:"basic"`
								ConventionalFuelSiteWithEv string `json:"conventional_fuel_site_with_ev"`
								DestinationChargingEv      string `json:"destination_charging_ev"`
								LngSite                    string `json:"lng_site"`
								MobilityHubEvHub           string `json:"mobility_hub_ev_hub"`
								MobilityHubEvPlusHub       string `json:"mobility_hub_ev_plus_hub"`
								NewFuelHub                 string `json:"new_fuel_hub"`
								NfrOnlySite                string `json:"nfr_only_site"`
								NfrSite                    string `json:"nfr_site"`
								Other                      string `json:"other"`
								Plural                     string `json:"plural"`
								Singular                   string `json:"singular"`
							} `json:"intro"`
							Offers   string `json:"offers"`
							Services struct {
								HydrogenSite         string `json:"hydrogen_site"`
								LngSite              string `json:"lng_site"`
								MobilityHubEvHub     string `json:"mobility_hub_ev_hub"`
								MobilityHubEvPlusHub string `json:"mobility_hub_ev_plus_hub"`
								NfrAgro              string `json:"nfr_agro"`
								Other                string `json:"other"`
								Plural               string `json:"plural"`
							} `json:"services"`
							Shop string `json:"shop"`
						} `json:"description"`
						DestinationHost string `json:"destination_host"`
						OpenStatus      struct {
							OpenNow           string `json:"open_now"`
							OpenNowUntilLater string `json:"open_now_until_later"`
							OpenNowUntilToday string `json:"open_now_until_today"`
						} `json:"open_status"`
						Sections struct {
							About struct {
								Title string `json:"title"`
							} `json:"about"`
							Details struct {
								Address    string `json:"address"`
								LatLng     string `json:"lat_lng"`
								Title      string `json:"title"`
								What3Words string `json:"what_3_words"`
							} `json:"details"`
							EvCharging struct {
								FurtherFees      string `json:"further_fees"`
								FurtherFeesTitle string `json:"further_fees_title"`
								LastUpdated      string `json:"last_updated"`
								OperatorNames    string `json:"operator_names"`
								PaymentOptions   struct {
									NewMotionApp string `json:"new_motion_app"`
									RfidToken    string `json:"rfid_token"`
									Title        string `json:"title"`
								} `json:"payment_options"`
								PaymentOptionsTitle string `json:"payment_options_title"`
								Points              string `json:"points"`
								PointsOutOf         string `json:"points_out_of"`
								Power               string `json:"power"`
								PowerAndCount       string `json:"power_and_count"`
								PricePerUnit        string `json:"price_per_unit"`
								Status              struct {
									Available   string `json:"available"`
									InUse       string `json:"in_use"`
									Unavailable string `json:"unavailable"`
									Unknown     string `json:"unknown"`
								} `json:"status"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"ev_charging"`
							Features struct {
								Title string `json:"title"`
							} `json:"features"`
							Footer struct {
								Links struct {
									Accessibility string `json:"accessibility"`
									Facebook      string `json:"facebook"`
									Instagram     string `json:"instagram"`
									LinkedIn      string `json:"linked_in"`
									NearMe        string `json:"near_me"`
									Privacy       string `json:"privacy"`
									SiteLocator   string `json:"site_locator"`
									Twitter       string `json:"twitter"`
									Youtube       string `json:"youtube"`
								} `json:"links"`
								Sections struct {
									MoreLocation string `json:"more_location"`
									MoreShell    string `json:"more_shell"`
									Social       string `json:"social"`
								} `json:"sections"`
							} `json:"footer"`
							FuelPricing struct {
								FuelPrices struct {
									Disclaimer string `json:"disclaimer"`
									NullPrice  string `json:"null_price"`
								} `json:"fuel_prices"`
								LastUpdated  string `json:"last_updated"`
								PricePerUnit string `json:"price_per_unit"`
								Title        string `json:"title"`
							} `json:"fuel_pricing"`
							Fuels struct {
								LastUpdated  string `json:"last_updated"`
								PricePerUnit string `json:"price_per_unit"`
								Title        string `json:"title"`
							} `json:"fuels"`
							Hydrogen struct {
								BusinessApp struct {
									HeavyDuty string `json:"heavy_duty"`
									LightDuty string `json:"light_duty"`
								} `json:"business_app"`
								NumberOfPumps string `json:"number_of_pumps"`
								PumpCount     string `json:"pump_count"`
								PumpName      struct {
									H35 string `json:"h35"`
									H70 string `json:"h70"`
								} `json:"pump_name"`
								PumpTypes        string `json:"pump_types"`
								SupportedVehicle struct {
									Bus   string `json:"bus"`
									Car   string `json:"car"`
									Truck string `json:"truck"`
									Van   string `json:"van"`
								} `json:"supported_vehicle"`
								SupportedVehicles string `json:"supported_vehicles"`
								Title             string `json:"title"`
								Unavailable       string `json:"unavailable"`
							} `json:"hydrogen"`
							Info struct {
								GetDirections  string `json:"get_directions"`
								Offers         string `json:"offers"`
								StationLocator string `json:"station_locator"`
							} `json:"info"`
							LocationDetails struct {
								Address    string `json:"address"`
								LatLng     string `json:"lat_lng"`
								Title      string `json:"title"`
								What3Words string `json:"what_3_words"`
							} `json:"location_details"`
							MoreAtLocation struct {
								Title string `json:"title"`
							} `json:"more_at_location"`
							Nearby struct {
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"nearby"`
							Offers struct {
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"offers"`
							OpeningHours struct {
								Carwash string `json:"carwash"`
								Days    struct {
									Fri string `json:"Fri"`
									Mon string `json:"Mon"`
									Sat string `json:"Sat"`
									Sun string `json:"Sun"`
									Thu string `json:"Thu"`
									Tue string `json:"Tue"`
									Wed string `json:"Wed"`
								} `json:"days"`
								Forecourt   string `json:"forecourt"`
								Hours       string `json:"hours"`
								NoHours     string `json:"no_hours"`
								Shop        string `json:"shop"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"opening_hours"`
							OpeningHoursEv struct {
								Days struct {
									Fri string `json:"Fri"`
									Mon string `json:"Mon"`
									Sat string `json:"Sat"`
									Sun string `json:"Sun"`
									Thu string `json:"Thu"`
									Tue string `json:"Tue"`
									Wed string `json:"Wed"`
								} `json:"days"`
								Ev          string `json:"ev"`
								Hours       string `json:"hours"`
								NoHours     string `json:"no_hours"`
								Title       string `json:"title"`
								Unavailable string `json:"unavailable"`
							} `json:"opening_hours_ev"`
							StaticMap struct {
								AltText      string `json:"alt_text"`
								GeomeAltText string `json:"geome_alt_text"`
							} `json:"static_map"`
						} `json:"sections"`
						SiteStatus struct {
							ClosedTemporarily string `json:"closed_temporarily"`
						} `json:"site_status"`
					} `json:"station_page"`
					TruckServices struct {
						AdblueTruck     string `json:"adblue_truck"`
						EuroshellCard   string `json:"euroshell_card"`
						GuardedCarPark  string `json:"guarded_car_park"`
						HeavyDutyEv     string `json:"heavy_duty_ev"`
						HgvLane         string `json:"hgv_lane"`
						MotorwayService string `json:"motorway_service"`
						ShellCard       string `json:"shell_card"`
						TruckParking    string `json:"truck_parking"`
						Truckport       string `json:"truckport"`
					} `json:"truck_services"`
					FuelLocalNames map[string]fuelLocalNames `json:"fuel_local_names"`
				} `json:"messages"`
				SupportedLocales []string `json:"supportedLocales"`
			} `json:"intlData"`
			LocalTpnEnabledCountries []string `json:"local_tpn_enabled_countries"`
			Properties               struct {
				Features struct {
					Enabled []string `json:"enabled"`
				} `json:"features"`
				Food struct {
					Enabled []string `json:"enabled"`
				} `json:"food"`
				MoreAtLocation struct {
					Enabled            []string `json:"enabled"`
					MaxStandaloneIcons int      `json:"maxStandaloneIcons"`
				} `json:"more_at_location"`
			} `json:"properties"`
			Sections struct {
				Footer struct {
					Links struct {
						Accessibility string `json:"accessibility"`
						Facebook      string `json:"facebook"`
						Instagram     string `json:"instagram"`
						LinkedIn      string `json:"linked_in"`
						NearMe        string `json:"near_me"`
						Privacy       string `json:"privacy"`
						SiteLocator   string `json:"site_locator"`
						Twitter       string `json:"twitter"`
						Youtube       string `json:"youtube"`
					} `json:"links"`
					SectionOrder []string `json:"section_order"`
					Sections     struct {
						MoreLocation []string `json:"more_location"`
						MoreShell    []string `json:"more_shell"`
						Social       []string `json:"social"`
					} `json:"sections"`
				} `json:"footer"`
			} `json:"sections"`
			StationPage struct {
				Description struct {
					FoodOfferings struct {
						Groups struct {
							Other []string `json:"other"`
						} `json:"groups"`
					} `json:"food_offerings"`
					Services struct {
						Groups struct {
							HydrogenSite []string `json:"hydrogen_site"`
							LngSite      []string `json:"lng_site"`
							NfrAgro      []string `json:"nfr_agro"`
							Other        []string `json:"other"`
						} `json:"groups"`
					} `json:"services"`
				} `json:"description"`
				Sections struct {
					About struct {
						Show bool `json:"show"`
					} `json:"about"`
					Details struct {
						Show string `json:"show"`
					} `json:"details"`
					EvCharging struct {
						Connectors struct {
							Enabled []string `json:"enabled"`
						} `json:"connectors"`
					} `json:"ev_charging"`
					Features struct {
						Enabled []string `json:"enabled"`
					} `json:"features"`
					Footer struct {
						Links struct {
							Accessibility string `json:"accessibility"`
							Facebook      string `json:"facebook"`
							Instagram     string `json:"instagram"`
							LinkedIn      string `json:"linked_in"`
							NearMe        string `json:"near_me"`
							Privacy       string `json:"privacy"`
							SiteLocator   string `json:"site_locator"`
							Twitter       string `json:"twitter"`
							Youtube       string `json:"youtube"`
						} `json:"links"`
						SectionOrder []string `json:"section_order"`
						Sections     struct {
							MoreLocation []string `json:"more_location"`
							MoreShell    []string `json:"more_shell"`
							Social       []string `json:"social"`
						} `json:"sections"`
					} `json:"footer"`
					FuelPricing struct {
						Enabled []string `json:"enabled"`
					} `json:"fuel_pricing"`
					LocationDetails struct {
						Show string `json:"show"`
					} `json:"location_details"`
					MoreAtLocation struct {
						Enabled []string `json:"enabled"`
					} `json:"more_at_location"`
				} `json:"sections"`
			} `json:"station_page"`
			Locale string `json:"locale"`
		} `json:"config"`
		Location struct {
			LocationID           string    `json:"location_id"`
			Name                 string    `json:"name"`
			Lat                  float64   `json:"lat"`
			Lng                  float64   `json:"lng"`
			FormattedAddress     string    `json:"formatted_address"`
			Telephone            string    `json:"telephone"`
			OpenStatus           string    `json:"open_status"`
			NextOpenStatusChange time.Time `json:"next_open_status_change"`
			FuelPricing          struct {
				Updated           string             `json:"updated"`
				Currency          string             `json:"currency"`
				Precision         int                `json:"precision"`
				Unit              string             `json:"unit"`
				CountryCode       string             `json:"country_code"`
				Prices            map[string]float32 `json:"prices"`
				Status            string             `json:"status"`
				SiteOperationType string             `json:"site_operation_type"`
				UnitOfPrice       int                `json:"unit_of_price"`
			} `json:"fuel_pricing"`
			CountryCode  string   `json:"country_code"`
			Amenities    []string `json:"amenities"`
			Fuels        []string `json:"fuels"`
			Section      []string `json:"section"`
			StaticMapURL string   `json:"static_map_url"`
			Offers       []struct {
				OfferID                 int    `json:"offer_id"`
				Title                   string `json:"title"`
				Description             string `json:"description"`
				ImageURL                string `json:"image_url"`
				ImageThumbnailMediumURL string `json:"image_thumbnail_medium_url"`
				ImageThumbnailSmallURL  string `json:"image_thumbnail_small_url"`
				VideoID                 string `json:"video_id"`
				Href                    string `json:"href"`
			} `json:"offers"`
			EvCharging struct {
				MaxPower       string   `json:"max_power"`
				ConnectorTypes []string `json:"connector_types"`
				AuthMethods    []any    `json:"auth_methods"`
				ConnectorData  []struct {
					Type      string `json:"type"`
					MaxPower  string `json:"max_power"`
					Status    string `json:"status"`
					Total     int    `json:"total"`
					Available int    `json:"available"`
					Pricing   []struct {
						Type       string `json:"type"`
						Currency   string `json:"currency"`
						EnergyFees struct {
							Unit  string `json:"unit"`
							Price string `json:"price"`
						} `json:"energy_fees"`
						Elements []struct {
							PriceComponents struct {
								Type         string `json:"type"`
								Price        string `json:"price"`
								Vat          string `json:"vat"`
								StepSize     string `json:"step_size"`
								Restrictions struct {
									StartTime   any    `json:"start_time"`
									EndTime     any    `json:"end_time"`
									StartDate   string `json:"start_date"`
									EndDate     string `json:"end_date"`
									MinKwh      any    `json:"min_kwh"`
									MaxKwh      any    `json:"max_kwh"`
									MinCurrent  any    `json:"min_current"`
									MaxCurrent  any    `json:"max_current"`
									MinPower    any    `json:"min_power"`
									MaxPower    any    `json:"max_power"`
									MinDuration int    `json:"min_duration"`
									MaxDuration any    `json:"max_duration"`
									DayOfWeek   []any  `json:"day_of_week"`
								} `json:"restrictions"`
							} `json:"price_components"`
						} `json:"elements"`
						SessionFeesStatement      string `json:"session_fees_statement"`
						BlockingFeesStatement     string `json:"blocking_fees_statement"`
						SubscriptionFeesStatement string `json:"subscription_fees_statement"`
						Comments                  string `json:"comments"`
						Providers                 []struct {
							ProviderID string `json:"provider_id"`
							PartyID    string `json:"party_id"`
							Role       string `json:"role"`
						} `json:"providers"`
					} `json:"pricing"`
				} `json:"connector_data"`
				ChargingPoints          int       `json:"charging_points"`
				CpoEmail                any       `json:"cpo_email"`
				CpoTelephoneNumber      string    `json:"cpo_telephone_number"`
				CpoWebsite              any       `json:"cpo_website"`
				OcpiEvseIds             []string  `json:"ocpi_evse_ids"`
				OperatorName            []string  `json:"operator_name"`
				EvseIds                 []string  `json:"evse_ids"`
				AvailableChargingPoints int       `json:"available_charging_points"`
				AvailabilityUpdated     time.Time `json:"availability_updated"`
				PricingUpdated          time.Time `json:"pricing_updated"`
			} `json:"ev_charging"`
			HydrogenOffering      any `json:"hydrogen_offering"`
			ForecourtOpeningHours []struct {
				Days  []string   `json:"days"`
				Hours [][]string `json:"hours"`
			} `json:"forecourt_opening_hours"`
			ShopOpeningHours         any    `json:"shop_opening_hours"`
			Description              string `json:"description"`
			ShopOpenStatus           string `json:"shop_open_status"`
			NextShopOpenStatusChange any    `json:"next_shop_open_status_change"`
			TzOffset                 int    `json:"tz_offset"`
			SiteStatus               string `json:"site_status"`
			CarwashOpeningHours      any    `json:"carwash_opening_hours"`
			EvOpeningHours           []struct {
				Days  []string   `json:"days"`
				Hours [][]string `json:"hours"`
			} `json:"ev_opening_hours"`
			EvOpenStatus           string `json:"ev_open_status"`
			NextEvOpenStatusChange any    `json:"next_ev_open_status_change"`
			DestinationHost        any    `json:"destination_host"`
		} `json:"location"`
		Links []struct {
			Text   string `json:"text"`
			Link   string `json:"link"`
			NewTab bool   `json:"newTab"`
		} `json:"links"`
		LocaleLinks []struct {
			Text   string `json:"text"`
			Link   string `json:"link"`
			Active bool   `json:"active"`
			NewTab bool   `json:"new_tab"`
		} `json:"localeLinks"`
		Breadcrumbs []struct {
			Text             string `json:"text"`
			Link             string `json:"link"`
			Active           bool   `json:"active"`
			HideHrefIfActive bool   `json:"hideHrefIfActive"`
		} `json:"breadcrumbs"`
		LocatorLink        string `json:"locatorLink"`
		DirectionsLinkHref string `json:"directionsLinkHref"`
		Nearby             []struct {
			ID                   string `json:"id"`
			Name                 string `json:"name"`
			FormattedAddress     string `json:"formatted_address"`
			OpenStatus           string `json:"open_status"`
			NextOpenStatusChange any    `json:"next_open_status_change"`
			Href                 string `json:"href"`
			TzOffset             int    `json:"tz_offset"`
			StaticMapURL         string `json:"static_map_url"`
		} `json:"nearby"`
	} `json:"props"`
	URL            string `json:"url"`
	Version        any    `json:"version"`
	EncryptHistory bool   `json:"encryptHistory"`
	ClearHistory   bool   `json:"clearHistory"`
}

func (s StationShell) Identifier() string {
	return s.url
}

func (s StationShell) ScrapePrices() (Sample, error) {
	req, err := http.NewRequest(http.MethodGet, s.url, nil)
	if err != nil {
		return Sample{}, err
	}

	//nolint:gosec // Insecure client is necessary here
	insecureTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	insecureClient := &http.Client{Transport: insecureTransport}

	var bytes []byte

	if err := retry.Do(context.TODO(), newScrapeRetry(), func(ctx context.Context) error {
		resp, err := insecureClient.Do(req)

		if err != nil {
			return retry.RetryableError(fmt.Errorf("could not complete request for price data: %w", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return retry.RetryableError(errors.New("request status was not '200 OK'"))
		}

		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("could not read price data from response body: %w", err))
		}

		return nil
	}); err != nil {
		return Sample{}, fmt.Errorf("station page request for station %s did not succeed after the maximum number of attempts (%d): %w", s.Identifier(), MAX_RETRIES, err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(bytes)))

	if err != nil {
		return Sample{}, err
	}

	reactPropsNode := htmlquery.FindOne(doc, `//script[@data-page]`)

	if reactPropsNode == nil || reactPropsNode.FirstChild == nil {
		return Sample{}, errors.New("could not find react-data-props for extraction")
	}

	propsString := reactPropsNode.FirstChild.Data
	var dataPage ShellDataPage
	err = json.Unmarshal([]byte(propsString), &dataPage)

	if err != nil {
		return Sample{}, err
	}

	result := Sample{
		Prices:      map[string]float32{},
		Time:        time.Now(),
		Address:     dataPage.Props.Location.FormattedAddress,
		GeoLocation: fmt.Sprintf("%f,%f", dataPage.Props.Location.Lat, dataPage.Props.Location.Lng),
		ScrapeID:    uuid.New(),
		Brand:       string(BrandShell),
	}

	for name, value := range dataPage.Props.Location.FuelPricing.Prices {
		translatedName := dataPage.Props.Config.IntlData.Messages.InfoWindow.Sections.Fuels.FuelLocalNames[name]["DE"]

		if len(strings.TrimSpace(translatedName)) == 0 {
			translatedName = dataPage.Props.Config.IntlData.Messages.InfoWindow.Sections.Fuels.FuelLocalNames[name]["other"]
		}

		result.Prices[translatedName] = float32(value)
	}

	return result, nil
}
