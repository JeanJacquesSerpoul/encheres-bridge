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

def has_control(hand, distribution, suit):
    """Vérifie si la main possède un contrôle (As, Roi, Singleton ou Chicane) dans la couleur."""
    # Contrôle de distribution : Chicane (0) ou Singleton (1)
    if distribution[suit] <= 1:
        return True
    # Contrôle d'honneurs : Contient l'As ou le Roi
    suit_ranks = [rank for s, rank in hand if s == suit]
    if 'A' in suit_ranks or 'K' in suit_ranks:
        return True
    return False

def generate_control_deal():
    """Génère une donne valide pour une séquence de contrôles initiée par Sud fitté à Pique."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur (Main forte de chelem)
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant (Soutien limite)
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture d'1 Pique forte (17-20 points H, au moins 5 Piques)
        if not (17 <= s_h <= 20): continue
        if s_dist['S'] < 5: continue
        
        # 2. Nord doit avoir une main de soutien limite (11-12 HL avec un fit 3/4ème à Pique)
        n_hl = n_h + (n_dist['S'] - 3 if n_dist['S'] > 3 else 0)
        if not (11 <= n_hl <= 12): continue
        if n_dist['S'] < 3: continue
        
        # Séquence de base fixée : 1S - Pass - 3S - Pass (Le fit Pique est agréé)
        # Sud a la parole et doit nommer son contrôle le plus économique au palier de 4.
        # L'ordre économique après 3♠ est : 4♣ -> 4♦ -> 4♥
        
        if has_control(sud, s_dist, 'C'):
            controle_sud = "4C"
            desc_ctrl = "Contrôle Trèfle (As, Roi, Singleton ou Chicane). C'est le contrôle le plus économique disponible."
        elif has_control(sud, s_dist, 'D'):
            controle_sud = "4D"
            desc_ctrl = "Contrôle Carreau. Sud a passé les Trèfles (dénie un contrôle Trèfle) pour nommer les Carreaux."
        elif has_control(sud, s_dist, 'H'):
            controle_sud = "4H"
            desc_ctrl = "Contrôle Cœur. Sud a passé les Trèfles et les Carreaux (pas de contrôle mineur) pour nommer les Cœurs."
        else:
            # Si Sud n'a aucun contrôle annexe, il ne peut pas explorer le chelem et doit conclure à 4♠
            controle_sud = "4S"
            desc_ctrl = "Pas de contrôle annexe disponible. Sud est obligé de conclure directement à la manche à 4♠."
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'controle_sud': controle_sud, 'desc_sud': desc_ctrl, 
            's_h': s_h, 'n_h': n_h
        }

def create_pbn_file(filename="encheres_controles.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux Enchères de Contrôles\n\n")
        
        for i in range(1, count + 1):
            deal = generate_control_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Contrôles - Donne {i}"]\n')
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
            # Séquence : Sud 1♠, Ouest Passe, Nord soutient à 3♠, Est Passe, Sud lance le premier contrôle
            f.write(f'1S Pass 3S Pass\n')
            f.write(f'{deal["controle_sud"]} Pass\n')
            f.write(f'[Note "Sud possède {deal["s_h"]}H et lance l\'exploration de chelem après le fit agréé."]\n')
            f.write(f'[Note "Enchère de Sud : {deal["controle_sud"]} -> {deal["desc_sud"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes d'Enchères de Contrôles souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"controles_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
