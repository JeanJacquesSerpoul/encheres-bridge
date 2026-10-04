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

def has_honor_stopper(hand, suit):
    """Vérifie si la main possède au moins un honneur majeur dans la couleur spécifiée."""
    suit_cards = [rank for s, rank in hand if s == suit]
    return any(h in suit_cards for h in ['A', 'K', 'Q', 'J'])

def format_pbn_player(hand):
    """Formate les cartes d'un joueur selon les spécifications PBN (S.H.D.C)."""
    suits = {'S': [], 'H': [], 'D': [], 'C': []}
    for suit, rank in hand:
        suits[suit].append(rank)
    
    order = "AKQJT98765432"
    for s in suits:
        suits[s].sort(key=lambda r: order.index(r))
        
    return ".".join(["".join(suits[s]) for s in ['S', 'H', 'D', 'C']])

def generate_reveil_1nt_deal():
    """Génère une donne valide pour un réveil à 1SA d'Est après une séquence 1H - P - P."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur
        ouest = deck[13:26]  # Le partenaire du réveilleur
        nord = deck[26:39]   # Le répondant adverse (qui a passé)
        est = deck[39:52]    # Le réveilleur en 1SA
        
        s_h, s_dist = eval_hand(sud)
        w_h, w_dist = eval_hand(ouest)
        n_h, n_dist = eval_hand(nord)
        e_h, e_dist = eval_hand(est)
        
        # 1. Sud doit avoir une ouverture légitime de 1 Cœur (1H)
        if not (12 <= s_h <= 14): continue
        if s_dist['H'] < 4 or s_dist['S'] >= 5: continue
        
        # Ouest passe légitimement en intervention directe (manque de points ou de couleur 5ème)
        if w_h > 11: continue
        
        # 2. Nord doit avoir un jeu faible (Moins de 5 points H) pour passer
        if n_h >= 5: continue
        
        # 3. Est est en position de réveil à 1SA sur 1 Coeur :
        # - Zone de points : 11 à 14 points H
        # - Distribution régulière (pas de chicane, pas de singleton, max une couleur 5ème mineure)
        if not (11 <= e_h <= 14): continue
        if any(v <= 1 or v >= 6 for v in e_dist.values()): continue
        if e_dist['S'] >= 5 or e_dist['H'] >= 5: continue
        
        # - Arrêt obligatoire à Cœur (couleur adverse)
        if e_dist['H'] < 2 or not has_honor_stopper(est, 'H'): continue
        
        # 4. Calcul de la réponse d'Ouest (le partenaire du réveilleur)
        # Face à un réveil de 1SA (11-14H), le répondant applique les mêmes conventions
        # que sur une ouverture de 1SA, mais en décalant les zones de points (il faut 11-12+ points pour la manche)
        if w_h <= 9 and w_dist['S'] < 5 and w_dist['H'] < 5:
            rep_ouest = "Pass"
            desc_ouest = "Passe avec un jeu faible et régulier (0-9H)"
        elif w_h >= 11:
            rep_ouest = "3NT"
            desc_ouest = "Conclusion à 3SA (Main forte de 11+H, la manche est acquise)"
        elif w_dist['S'] >= 5:
            rep_ouest = "2D"  # Exemple de Texas pour les Piques si on joue le même système
            desc_ouest = "Utilisation d'un Texas Pique (2♦) ou d'un Stayman selon les conventions de la paire"
        else:
            rep_ouest = "2NT"
            desc_ouest = "Proposition de manche à 2SA (Zone limite de 10H)"
            
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'rep_ouest': rep_ouest, 'desc_ouest': desc_ouest, 'e_h': e_h, 'w_h': w_h
        }

def create_pbn_file(filename="reveil_1sa.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement au Réveil par 1SA\n\n")
        
        for i in range(1, count + 1):
            deal = generate_reveil_1nt_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Reveil 1SA - Donne {i}"]\n')
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
            # Séquence fixe : Sud ouvre d'1H, Ouest passe, Nord passe, Est réveille à 1SA
            f.write(f'1H Pass Pass 1NT\n')
            f.write(f'Pass {deal["rep_ouest"]} Pass\n') 
            f.write(f'[Note "Est reveille a 1SA avec une main reguliere de {deal["e_h"]}H et arret Coeur."]\n')
            f.write(f'[Note "Ouest possède {deal["w_h"]}H et choisit : {deal["desc_ouest"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur le Réveil par 1SA souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"reveil_1sa_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
