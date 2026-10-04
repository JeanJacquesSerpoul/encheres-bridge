import random

# Valeur des honneurs au bridge
HCP = {'A': 4, 'K': 3, 'Q': 2, 'J': 1, 'T': 0, '9': 0, '8': 0, '7': 0, '6': 0, '5': 0, '4': 0, '3': 0, '2': 0}

def eval_hand(hand):
    """Calcule les points d'honneurs (H) et la distribution d'une main."""
    distribution = {'S': 0, 'H': 0, 'D': 0, 'C': 0}
    for suit, rank in hand:
        distribution[suit] += 1
    h_points = sum(HCP[rank] for _, rank in hand)
    return h_points, distribution

def format_pbn_player(hand):
    """Formate les cartes d'un joueur selon les spécifications PBN (S.H.D.C)."""
    suits = {'S': [], 'H': [], 'D': [], 'C': []}
    for suit, rank in hand:
        suits[suit].append(rank)
    
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_valid_fourth_forcing_deal():
    """Génère une donne fluide propice à une séquence en 4ème Couleur Forcing."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]
        nord = deck[26:39]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Critères pour Sud (L'ouvreur) : Main régulière ou bicolore classique (12-14H ou 15-17H léger)
        if not (12 <= s_h <= 16): continue
        if s_dist['C'] < 4 and s_dist['D'] < 4: continue # Doit ouvrir d'une mineure
        
        # 2. Critères pour Nord (Le répondant) : 11-14 HL (Espoir de manche ou manche)
        if not (11 <= n_h <= 14): continue
        
        # On force une distribution type : Sud ouvre en mineure, Nord nomme des Coeurs
        if s_dist['D'] >= s_dist['C']:
            ouv_sud = "1D"
        else:
            ouv_sud = "1C"
            
        # Nord doit avoir au moins 5 cartes à Coeur pour vouloir vérifier un fit 3ème plus tard
        if n_dist['H'] != 5: continue
        rep_nord = "1H"
        
        # Sud fait une redemande naturelle à 1 Pique (il a au moins 4 Piques, et pas de fit 4ème à Coeur)
        if s_dist['S'] < 4 or s_dist['H'] >= 4: continue
        red_sud = "1S"
        
        # Détermination de la 4ème couleur artificielle disponible (ici les Carreaux ou les Trèfles)
        # Séquence : 1Min - 1H - 1S -> la 4ème couleur est l'autre mineure !
        quatrieme_couleur = "2C" if ouv_sud == "1D" else "2D"
        
        ouest = deck[13:26]
        est = deck[39:52]
        
        # Analyse de la réponse que devra faire Sud à la 4ème couleur forcing :
        # - Donner le fit 3ème à Coeur s'il l'a
        # - Promettre un arrêt dans la 4ème couleur en disant 2SA/3SA
        # - Répéter sa couleur d'ouverture par défaut
        if s_dist['H'] == 3:
            prev_rep = "2H / 3H"
            desc_reponse = "Fit 3ème dans les Cœurs du partenaire"
        elif any(r in [c[1] for c in sud if c[0] == quatrieme_couleur[1]] for r in ['A', 'K', 'Q']):
            prev_rep = "2NT / 3NT"
            desc_reponse = "Pas de fit, mais tient bien la 4ème couleur (Arrêt garanti)"
        else:
            prev_rep = "Enchère par défaut"
            desc_reponse = "Pas de fit à Coeur et pas d'arrêt solide dans la 4ème couleur"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'encheres': [ouv_sud, rep_nord, red_sud, quatrieme_couleur],
            'desc_reponse': desc_reponse, 'prev_rep': prev_rep
        }

def create_pbn_file(filename="quatrieme_forcing.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement à la 4ème Couleur Forcing\n\n")
        
        for i in range(1, count + 1):
            deal = generate_valid_fourth_forcing_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement 4eme Couleur Forcing - Donne {i}"]\n')
            f.write('[Date "2026.10.04"]\n')
            f.write(f'[Board "{i}"]\n')
            f.write('[West "-"]\n')
            f.write('[North "-"]\n')
            f.write('[East "-"]\n')
            f.write('[South "-"]\n')
            f.write('[Dealer "S"]\n')
            f.write('[Vulnerable "None"]\n')
            f.write(f'[Deal "{pbn_deal}"]\n')
            f.write('[Scoring "MP"]\n')
            f.write('[Auction "S"]\n')
            # Déroulement automatique de la séquence jusqu'à la 4ème couleur forcing de Nord
            f.write(f'{deal["encheres"][0]} Pass {deal["encheres"][1]} Pass\n')
            f.write(f'{deal["encheres"][2]} Pass {deal["encheres"][3]} Pass\n')
            f.write(f'[Note "Nord a enchérit la 4ème couleur ({deal["encheres"][3]}) artificielle et forcing."]\n')
            f.write(f'[Note "Sud doit répondre selon son jeu : {deal["desc_reponse"]} ({deal["prev_rep"]})"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de 4ème Couleur Forcing souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"4eme_forcing_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
