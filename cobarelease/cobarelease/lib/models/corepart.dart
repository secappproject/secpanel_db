class Corepart {
  int? id;
  String panelNoPp;
  String vendor;
  Corepart({this.id, required this.panelNoPp, required this.vendor});

  Map<String, dynamic> toMap() {
    return {'id': id, 'panel_no_pp': panelNoPp, 'vendor': vendor};
  }

  factory Corepart.fromMap(Map<String, dynamic> map) {
    return Corepart(
      id: map['id'],
      panelNoPp: map['panel_no_pp'],
      vendor: map['vendor'],
    );
  }
}
