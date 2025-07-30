import 'package:secpanel/models/approles.dart';

class Company {
  String id;
  // String password;
  String name;
  AppRole role;
  Company({
    required this.id,
    // required this.password,
    required this.name,
    required this.role,
  });

  Map<String, dynamic> toMap() {
    return {
      'id': id,
      // 'password': password,
      'name': name, 'role': role.name,
    };
  }

  factory Company.fromMap(Map<String, dynamic> map) {
    return Company(
      id: map['id'],
      // password: map['password'],
      name: map['name'],
      role: AppRole.values.firstWhere((e) => e.name == map['role']),
    );
  }
}
