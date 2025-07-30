class Component {
  int? id;
  String panelNoPp; // Foreign Key ke Panel.noPp
  String vendor; // Foreign Key ke User.id
  Component({this.id, required this.panelNoPp, required this.vendor});

  Map<String, dynamic> toMap() {
    return {'id': id, 'panel_no_pp': panelNoPp, 'vendor': vendor};
  }

  factory Component.fromMap(Map<String, dynamic> map) {
    return Component(
      id: map['id'],
      panelNoPp: map['panel_no_pp'],
      vendor: map['vendor'],
    );
  }
}
