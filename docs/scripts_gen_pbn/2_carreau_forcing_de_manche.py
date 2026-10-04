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

def count_losers(hand, distribution):
    """Estimation simplifiée des perdantes dans les couleurs longues."""
    losers = 0
    # Groupement des cartes par couleur
    suits = {'S': [], 'H': [], 'D': [], 'C': []}
    for suit, rank in hand:
        suits[suit].append(rank)
    
    for suit, cards in suits.items():
        L = len(cards)
        if L == 0:
            continue
        # On regarde les 3 premières cartes de la couleur
        top_cards = cards[:3]
        needed = min(L, 3)
        has_honors = sum(1 for r in top_cards if r in ['A', 'K', 'Q'])
        losers += (needed - has_honors)
    return losers

def format_pbn_player(hand):
    """Formate les cartes d'un joueur selon les spécifications PBN (S.H.D.C)."""
    suits = {'S': [], 'H': [], 'D': [], 'C': []}
    for suit, rank in hand:
        suits[suit].append(rank)
    
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_valid_forcing_manche_deal():
    """Génère une distribution qui valide les critères d'un 2 Carreaux Forcing de Manche (Sud)."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]
        s_h, s_dist = eval_hand(sud)
        
        # Condition 1 : Soit 24+ points H (Main régulière géante)
        # Condition 2 : Soit unicolore fort (20+ points H ET max 2 perdantes globales)
        is_regular_giant = (s_h >= 24)
        
        is_strong_unicolor = False
        if s_h >= 20 and any(v >= 6 for v in s_dist.values()):
            if count_losers(sud, s_dist) <= 2:
                is_strong_unicolor = True
                
        if not (is_regular_giant or is_strong_unicolor): 
            continue
        
        ouest = deck[13:26]
        nord = deck[26:39]
        est = deck[39:52]
        
        n_h, n_dist = eval_hand(nord)
        
        # Réponse automatique standard de Nord (Le répondant) selon le SEF :
        # 2 Coeurs (2H) est le relais obligatoire (artificiel et négatif), quelles que soient ses cartes.
        # (Certains systèmes utilisent des réponses optimisées au Roi, mais 2H reste la base absolue d'attente).
        rep_nord = "2H"
        desc_nord = "Relais obligatoire et automatique (Négatif ou d'attente)"
        
        if is_regular_giant:
            desc_sud = f"Ouverture de 2♦ Forcing de Manche - Main Regulière ({s_h}H)"
        else:
            longue = [k for k, v in s_dist.items() if v == max(s_dist.values())][0]
            desc_sud = f"Ouverture de 2♦ Forcing de Manche - Unicolore {longue} ({s_h}H, très peu de perdantes)"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture': "2D", 'rep_nord': rep_nord,
            'desc_sud': desc_sud, 'desc_nord': desc_nord
        }

def create_pbn_file(filename="deux_carreaux_fm.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'ouverture de 2♦ Forcing de Manche\n\n")
        
        for i in range(1, count + 1):
            deal = generate_valid_forcing_manche_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement 2 Carreaux FM - Donne {i}"]\n')
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
            f.write(f'{deal["ouverture"]} Pass {deal["rep_nord"]} Pass\n') # S: 2♦, N: 2♥ (relais)
            f.write(f'[Note "Sud montre: {deal["desc_sud"]}"]\n')
            f.write(f'[Note "Nord repond: {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes de 2♦ Forcing de Manche souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"deux_carreaux_fm_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
