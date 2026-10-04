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
    
    # Tri décroissant des cartes (A, K, Q... 2)
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_valid_two_weak_deal():
    """Génère une distribution de 52 cartes qui valide les critères d'un 2 faible majeur (Sud)."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]
        s_h, s_dist = eval_hand(sud)
        
        # 1. Zone de points H pour un 2 faible : 6 à 10 H
        if not (6 <= s_h <= 10): continue
        
        # Choix aléatoire de la majeure d'ouverture (S = Pique, H = Coeur)
        majeure = random.choice(['S', 'H'])
        
        # 2. Critères de distribution de l'ouvreur
        if s_dist[majeure] != 6: continue # Couleur strictement 6ème
        
        # Éviter une autre majeure 4ème ou + (ce qui fausserait l'ouverture)
        autre_maj = 'H' if majeure == 'S' else 'S'
        if s_dist[autre_maj] >= 4: continue
        
        # Éviter une main trop distribuée (bicolore 6-6 ou couleur annexe trop longue)
        if any(v >= 6 for k, v in s_dist.items() if k != majeure): continue
        
        ouest = deck[13:26]
        nord = deck[26:39]
        est = deck[39:52]
        
        # Enchère d'ouverture correspondante
        ouverture = "2S" if majeure == 'S' else "2H"
        nom_majeure = "Piques" if majeure == 'S' else "Coeurs"
        desc = f"Ouverture de 2 {nom_majeure} faible ({s_h}H, 6 cartes)"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture': ouverture, 'desc': desc
        }

def create_pbn_file(filename="deux_faible_entrainement.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'ouverture de 2 faible majeur\n\n")
        
        for i in range(1, count + 1):
            deal = generate_valid_two_weak_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement 2 Faible - Donne {i}"]\n')            
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
            f.write(f'{deal["ouverture"]} Pass\n') # Sud ouvre de 2 faible, Ouest passe
            f.write(f'[Note "Sud montre: {deal["desc"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de 2 faible souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"deux_faible_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
