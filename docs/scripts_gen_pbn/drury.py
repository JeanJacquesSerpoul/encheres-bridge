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

def generate_drury_deal():
    """Génère une donne valide pour la convention Drury."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur en 3ème position
        ouest = deck[13:26]  # Le flanc Ouest
        nord = deck[26:39]   # Le répondant (qui va faire le Drury)
        est = deck[39:52]    # Le flanc Est
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        n_h, n_dist = eval_hand(nord)
        e_h, e_dist = eval_hand(est)
        
        # 1. Sud et Nord doivent impérativement PASSER d'entrée
        if s_h >= 12 or n_h >= 12: continue
        if w_h >= 12: continue # On évite une ouverture d'Ouest pour simplifier la donne en 3ème position
        
        # 2. Sud doit posséder une ouverture en 3ème position en majeure (11-13 HL)
        majeure_sud = random.choice(['S', 'H'])
        s_hl = s_h + (s_dist[majeure_sud] - 4 if s_dist[majeure_sud] > 4 else 0)
        if not (11 <= s_hl <= 13): continue
        if s_dist[majeure_sud] < 5: continue # Majeure 5ème obligatoire
        
        # 3. Nord doit avoir un fit 3ème ou 4ème et une zone limite de Drury (9-11 HL)
        n_hl = n_h + (n_dist[majeure_sud] - 3 if n_dist[majeure_sud] > 3 else 0)
        if not (9 <= n_hl <= 11): continue
        if n_dist[majeure_sud] < 3: continue # Fit 3ème minimum requis
        
        # 4. Calcul de la réponse de Sud (Ouvreur) au Drury (2♣) de son partenaire
        # Selon le Drury fitté standard (SEF) :
        # - 2♦ montre une ouverture "légère" de 3ème position (11-12 HL). Demande l'arrêt.
        # - La répétition de la majeure (2♥ ou 2♠) montre une ouverture normale/forte (13+ HL).
        if s_hl <= 12:
            rep_sud = "2D"
            desc_sud = f"Ouverture légère ({s_hl} HL). Signale son manque de force par 2♦."
        else:
            rep_sud = f"2{majeure_sud}"
            desc_sud = f"Ouverture normale/forte ({s_hl} HL). Confirme sa force en répétant sa majeure."
            
        ouverture_sud = f"1{majeure_sud}"
        
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture_sud': ouverture_sud, 'rep_sud': rep_sud,
            'desc_sud': desc_sud, 's_hl': s_hl, 'n_hl': n_hl, 'majeure': majeure_sud
        }

def create_pbn_file(filename="drury_entrainement.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au DRURY\n\n")
        
        for i in range(1, count + 1):
            deal = generate_drury_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Drury - Donne {i}"]\n')
            f.write('[Date "2026.10.04"]\n')
            f.write(f'[Board "{i}"]\n')
            f.write('[West "-"]\n')
            f.write('[North "-"]\n')
            f.write('[East "-"]\n')
            f.write('[South "-"]\n')
            f.write('[Dealer "N"]\n') # Nord est le premier donneur et passe
            f.write('[Vulnerable "None"]\n')
            f.write(f'[Deal "{pbn_deal}"]\n')
            f.write('[Scoring "MP"]\n')
            f.write('[Auction "N"]\n')
            # Séquence : Nord Passe, Ouest Passe, Sud ouvre en 3ème, Est Passe, Nord dit 2♣ (Drury)
            f.write(f'Pass Pass {deal["ouverture_sud"]} Pass\n')
            f.write(f'2C Pass {deal["rep_sud"]} Pass\n') 
            f.write(f'[Note "Nord utilise le Drury (2♣) avec un fit {deal["majeure"]} et {deal["n_hl"]} HL."]\n')
            f.write(f'[Note "Sud possède {deal["s_hl"]} HL et répond : {deal["desc_sud"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur la convention Drury souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"drury_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
