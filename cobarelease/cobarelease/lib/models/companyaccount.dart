// file: models/company_account.dart
class CompanyAccount {
  String username;
  String password;
  String companyId;

  CompanyAccount({
    required this.username,
    required this.password,
    required this.companyId,
  });

  Map<String, dynamic> toMap() {
    return {
      'username': username,
      'password': password,
      'company_id': companyId,
    };
  }

  factory CompanyAccount.fromMap(Map<String, dynamic> map) {
    return CompanyAccount(
      username: map['username'],
      password: map['password'],
      companyId: map['company_id'],
    );
  }
}
