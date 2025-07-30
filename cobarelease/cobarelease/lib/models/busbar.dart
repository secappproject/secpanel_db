class Busbar {
  int? id;
  String panelNoPp;
  String vendor;
  String? remarks;

  Busbar({
    this.id,
    required this.panelNoPp,
    required this.vendor,
    this.remarks,
  });

  Map<String, dynamic> toMap() {
    return {
      'id': id,
      'panel_no_pp': panelNoPp,
      'vendor': vendor,
      'remarks': remarks,
    };
  }

  factory Busbar.fromMap(Map<String, dynamic> map) {
    return Busbar(
      id: map['id'],
      panelNoPp: map['panel_no_pp'],
      vendor: map['vendor'],
      remarks: map['remarks'],
    );
  }
}
