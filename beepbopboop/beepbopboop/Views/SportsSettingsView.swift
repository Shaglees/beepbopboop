import SwiftUI

private struct SportsTeam {
    let name: String
    let abbr: String
}

private struct SportsLeague {
    let name: String
    let id: String
    let teams: [SportsTeam]
}

private let allLeagues: [SportsLeague] = [
    SportsLeague(name: "NBA", id: "nba", teams: [
        SportsTeam(name: "Boston Celtics", abbr: "bos"),
        SportsTeam(name: "Chicago Bulls", abbr: "chi"),
        SportsTeam(name: "Denver Nuggets", abbr: "den"),
        SportsTeam(name: "Golden State Warriors", abbr: "gsw"),
        SportsTeam(name: "Los Angeles Lakers", abbr: "lal"),
        SportsTeam(name: "Miami Heat", abbr: "mia"),
        SportsTeam(name: "Milwaukee Bucks", abbr: "mil"),
        SportsTeam(name: "New York Knicks", abbr: "nyk"),
        SportsTeam(name: "Oklahoma City Thunder", abbr: "okc"),
        SportsTeam(name: "San Antonio Spurs", abbr: "sas"),
    ]),
    SportsLeague(name: "NHL", id: "nhl", teams: [
        SportsTeam(name: "Boston Bruins", abbr: "bos"),
        SportsTeam(name: "Calgary Flames", abbr: "cgy"),
        SportsTeam(name: "Chicago Blackhawks", abbr: "chi"),
        SportsTeam(name: "Edmonton Oilers", abbr: "edm"),
        SportsTeam(name: "Montreal Canadiens", abbr: "mtl"),
        SportsTeam(name: "New York Rangers", abbr: "nyr"),
        SportsTeam(name: "Ottawa Senators", abbr: "ott"),
        SportsTeam(name: "Toronto Maple Leafs", abbr: "tor"),
        SportsTeam(name: "Vancouver Canucks", abbr: "van"),
        SportsTeam(name: "Winnipeg Jets", abbr: "wpg"),
    ]),
    SportsLeague(name: "MLB", id: "mlb", teams: [
        SportsTeam(name: "Atlanta Braves", abbr: "atl"),
        SportsTeam(name: "Boston Red Sox", abbr: "bos"),
        SportsTeam(name: "Chicago Cubs", abbr: "chc"),
        SportsTeam(name: "Houston Astros", abbr: "hou"),
        SportsTeam(name: "Los Angeles Dodgers", abbr: "lad"),
        SportsTeam(name: "New York Yankees", abbr: "nyy"),
        SportsTeam(name: "San Francisco Giants", abbr: "sfg"),
        SportsTeam(name: "St. Louis Cardinals", abbr: "stl"),
        SportsTeam(name: "Toronto Blue Jays", abbr: "tor"),
        SportsTeam(name: "Cincinnati Reds", abbr: "cin"),
    ]),
    SportsLeague(name: "NFL", id: "nfl", teams: [
        SportsTeam(name: "Buffalo Bills", abbr: "buf"),
        SportsTeam(name: "Chicago Bears", abbr: "chi"),
        SportsTeam(name: "Dallas Cowboys", abbr: "dal"),
        SportsTeam(name: "Denver Broncos", abbr: "den"),
        SportsTeam(name: "Green Bay Packers", abbr: "gb"),
        SportsTeam(name: "Kansas City Chiefs", abbr: "kc"),
        SportsTeam(name: "New England Patriots", abbr: "ne"),
        SportsTeam(name: "Philadelphia Eagles", abbr: "phi"),
        SportsTeam(name: "San Francisco 49ers", abbr: "sf"),
        SportsTeam(name: "Seattle Seahawks", abbr: "sea"),
    ]),
]

struct SportsSettingsView: View {
    @Binding var followedTeams: Set<String>

    var body: some View {
        List {
            ForEach(allLeagues, id: \.id) { league in
                Section(league.name) {
                    ForEach(league.teams, id: \.name) { team in
                        let key = "\(league.id):\(team.abbr)"
                        Button {
                            if followedTeams.contains(key) {
                                followedTeams.remove(key)
                            } else {
                                followedTeams.insert(key)
                            }
                        } label: {
                            HStack {
                                Text(team.name)
                                    .foregroundColor(.primary)
                                Spacer()
                                if followedTeams.contains(key) {
                                    Image(systemName: "checkmark")
                                        .foregroundColor(.accentColor)
                                }
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Sports & Teams")
        .navigationBarTitleDisplayMode(.inline)
    }
}
