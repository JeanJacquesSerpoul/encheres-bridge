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

def generate_sputnik_deal():
    """Génère une donne valide pour un contre Sputnik de Nord."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur
        ouest = deck[13:26]  # L'intervenant adverse
        nord = deck[26:39]   # Le répondant (Contreur Sputnik)
        est = deck[39:52]    # Le partenaire de l'intervenant
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud ouvre d'1 Carreau (1D) régulier ou bicolore (12-14H)
        if not (12 <= s_h <= 14): continue
        if s_dist['D'] < 4 or s_dist['S'] >= 5 or s_dist['H'] >= 5: continue
        ouv_sud = "1D"
        
        # 2. Ouest fait une intervention naturelle à 1 Cœur (1H) (8-14H, 5 cartes)
        if not (8 <= w_h <= 14): continue
        if w_dist['H'] < 5: continue
        interv_ouest = "1H"
        
        # 3. Nord a du jeu (7-11H) et possède EXACTEMENT 4 cartes à Pique (la majeure restante)
        # S'il en avait 5, il dirait "1 Pique" naturellement. S'il en avait moins, il passerait ou dirait SA.
        if not (7 <= n_h <= 11): continue
        if n_dist['S'] != 4: continue
        
        # Le contre Sputnik est validé !
        rep_nord = "X" 
        
        # 4. Calcul de la redemande de l'ouvreur (Sud) après le Contre Sputnik de son partenaire :
        # - Si Sud a 4 cartes à Pique -> il fitte immédiatement (2♠ ou 3♠ selon sa force)
        # - Si Sud n'a pas 4 cartes à Pique -> il propose Sans-Atout avec un arrêt Cœur ou répète sa mineure
        if s_dist['S'] == 4:
            red_sud = "2S"
            desc_sud = "Fit trouvé ! Sud soutient à 2♠ (minimum) car le contre a promis 4 cartes à Pique."
        elif any(r in [c for c in sud if c == 'H'] for r in ['A', 'K', 'Q', 'J']):
            red_sud = "1NT"
            desc_sud = "Pas de fit Pique, mais Sud possède un arrêt Cœur et rechante à 1SA."
        else:
            red_sud = "2D"
            desc_sud = "Pas de fit Pique, pas d'arrêt Cœur franc. Sud répète ses Carreaux par défaut."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouv_sud': ouv_sud, 'interv_ouest': interv_ouest, 'rep_nord': rep_nord,
            'red_sud': red_sud, 'desc_sud': desc_sud, 'n_h': n_h, 's_h': s_h
        }

def create_pbn_file(filename="contre_sputnik.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au Contre Sputnik\n\n")
        
        for i in range(1, count + 1):
            deal = generate_sputnik_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Contre Sputnik - Donne {i}"]\n')
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
            # Séquence : Sud ouvre, Ouest intervient, Nord Contre (X), Est passe, Sud redemande
            f.write(f'{deal["ouv_sud"]} {deal["interv_ouest"]} {deal["rep_nord"]} Pass\n')
            f.write(f'{deal["red_sud"]} Pass\n')
            f.write(f'[Note "Nord contre Sputnik avec {deal["n_h"]}H, garantissant exactement 4 cartes à Pique."]\n')
            f.write(f'[Note "Sud possède {deal["s_h"]}H et redemande : {deal["desc_sud"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur le Contre Sputnik souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"contre_sputnik_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
