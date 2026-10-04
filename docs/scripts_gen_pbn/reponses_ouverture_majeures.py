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

def generate_response_major_deal():
    """Génère une donne valide où Sud ouvre d'1 en majeure et Nord applique le système de réponses."""
    deck = [(suit, rank) for suit in ['S', 'H', 'D', 'C'] for rank in "AKQJT98765432"]
    
    while True:
        random.shuffle(deck)
        sud = deck[0:13]    # L'ouvreur
        ouest = deck[13:26]
        nord = deck[26:39]   # Le répondant à tester
        est = deck[39:52]
        
        s_h, s_dist = eval_hand(sud)
        n_h, n_dist = eval_hand(nord)
        
        # 1. Sud doit avoir une ouverture légitime d'1 en majeure (12-14H standard, couleur 5ème+)
        if not (12 <= s_h <= 14): continue
        if s_dist['S'] < 5 and s_dist['H'] < 5: continue
        
        # Choix de la majeure (la plus longue)
        ouv_suit = 'S' if s_dist['S'] >= s_dist['H'] else 'H'
        ouverture_sud = f"1{ouv_suit}"
        
        # On calcule les points de Nord (Honneurs + Longueur éventuelle si fitté)
        # En standard SEF, la valorisation HL donne +1 pour le fit 3ème, +2 pour le fit 4ème
        n_hl = n_h
        if n_dist[ouv_suit] == 3: n_hl += 1
        elif n_dist[ouv_suit] >= 4: n_hl += 2
        
        # Nord passe s'il a moins de 6 points
        if n_hl < 6:
            rep_nord, desc = "Pass", "Passe par manque de jeu (Moins de 6 points)"
            
        # 2. Calcul automatique de la meilleure réponse de Nord (SEF)
        # Cas A : Nord est fitté (3+ cartes)
        elif n_dist[ouv_suit] >= 3:
            if 6 <= n_hl <= 10:
                rep_nord, desc = f"2{ouv_suit}", f"Soutien simple fitté (Zone minimum, 6-10 HL)"
            elif 11 <= n_hl <= 12:
                rep_nord, desc = f"3{ouv_suit}", f"Soutien limite invitatif (Zone intermédiaire, 11-12 HL)"
            else:
                rep_nord, desc = f"4{ouv_suit}", f"Conclusion à la manche (Zone forte de manche, 13+ HL ou distribution)"
                
        # Cas B : Nord n'est pas fitté (0-2 cartes)
        else:
            # Cas particulier : Ouverture d'1 Cœur et Nord a 4 Piques
            if ouv_suit == 'H' and n_dist['S'] >= 4:
                rep_nord, desc = "1S", "Changement de couleur à l'économie à 1♠ (4+ cartes, dès 6H)"
            elif n_h >= 11:
                # Changement de couleur au niveau de 2 (On cherche la mineure la plus longue)
                pref_min = 'C' if n_dist['C'] >= n_dist['D'] else 'D'
                rep_nord, desc = f"2{pref_min}", f"Changement de couleur au niveau de 2 forcing pour un tour (11+H, couleur {pref_min})"
            else:
                rep_nord, desc = "1NT", "Enchère par défaut à 1SA (Pas de fit, pas de couleur au niveau 1, zone 6-10H)"
                
        return {
            'S': sud, 'W': ouest, 'N': nord, 'E': est,
            'ouverture_sud': ouverture_sud, 'rep_nord': rep_nord, 'desc_nord': desc, 
            's_h': s_h, 'n_h': n_h, 'ouv_suit': ouv_suit
        }

def create_pbn_file(filename="reponses_majeures.pbn", count=50):
    """Génère le fichier au format .PBN avec le nombre de donnes paramétré."""
    with open(filename, "w", encoding="utf-8") as f:
        f.write("% PBN 2.1\n")
        f.write(f"% Génération automatique de {count} donnes pour l'entraînement aux réponses sur ouverture majeure\n\n")
        
        for i in range(1, count + 1):
            deal = generate_response_major_deal()
            pbn_deal = f"S:{format_pbn_player(deal['S'])} {format_pbn_player(deal['W'])} {format_pbn_player(deal['N'])} {format_pbn_player(deal['E'])}"
            
            f.write(f'[Event "Entrainement Réponses Majeures - Donne {i}"]\n')
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
            # Séquence : Sud ouvre d'1 en majeure, Ouest Passe, Nord produit sa réponse calculée
            f.write(f'{deal["ouverture_sud"]} Pass {deal["rep_nord"]} Pass\n')
            f.write(f'[Note "Sud ouvre de {deal["ouverture_sud"]} ({deal["s_h"]}H, majeure 5ème)."]\n')
            f.write(f'[Note "Nord possède {deal["n_h"]}H et doit répondre : {deal["desc_nord"]}"]\n\n')
            
    print(f"\nFélicitations ! Le fichier '{filename}' contenant {count} donnes a été créé avec succès.")

if __name__ == "__main__":
    try:
        saisie = input("Combien de donnes sur les réponses en majeure souhaitez-vous générer ? (Par défaut : 50) : ")
        if saisie.strip() == "":
            NOMBRE_DE_DONNES = 50
        else:
            NOMBRE_DE_DONNES = int(saisie)
    except ValueError:
        print("Saisie invalide. Génération par défaut de 50 donnes.")
        NOMBRE_DE_DONNES = 50

    # Lancement de la création
    create_pbn_file(filename=f"reponses_majeures_{NOMBRE_DE_DONNES}_donnes.pbn", count=NOMBRE_DE_DONNES)
