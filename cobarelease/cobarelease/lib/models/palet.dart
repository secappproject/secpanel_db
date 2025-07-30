class Palet {
  int? id;
  String panelNoPp;
  String vendor;
  Palet({this.id, required this.panelNoPp, required this.vendor});

  Map<String, dynamic> toMap() {
    return {'id': id, 'panel_no_pp': panelNoPp, 'vendor': vendor};
  }

  factory Palet.fromMap(Map<String, dynamic> map) {
    return Palet(
      id: map['id'],
      panelNoPp: map['panel_no_pp'],
      vendor: map['vendor'],
    );
  }
}
