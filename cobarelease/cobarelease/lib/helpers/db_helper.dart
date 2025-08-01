import 'dart:convert';
import 'dart:io';
import 'package:excel/excel.dart';
import 'package:path/path.dart';
import 'package:path_provider/path_provider.dart';
import 'package:sqflite/sqflite.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/busbar.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/models/companyaccount.dart';
import 'package:secpanel/models/component.dart';
import 'package:secpanel/models/corepart.dart';
import 'package:secpanel/models/palet.dart';
import 'package:secpanel/models/paneldisplaydata.dart';
import 'package:secpanel/models/panels.dart';

class TemplateFile {
  final List<int> bytes;
  final String extension;
  TemplateFile({required this.bytes, required this.extension});
}

class DatabaseHelper {
  DatabaseHelper._privateConstructor();
  static final DatabaseHelper instance = DatabaseHelper._privateConstructor();

  static Database? _database;
  Future<Database> get database async => _database ??= await _initDatabase();

  Future<Database> _initDatabase() async {
    Directory documentsDirectory = await getApplicationDocumentsDirectory();
    String path = join(documentsDirectory.path, 'app_database_final_4.db');
    return await openDatabase(
      path,
      version: 2,
      onConfigure: (db) async {
        await db.execute('PRAGMA foreign_keys = ON');
      },
      onCreate: _onCreate,
      onUpgrade: (db, oldVersion, newVersion) async {
        var batch = db.batch();
        batch.execute('DROP TABLE IF EXISTS components');
        batch.execute('DROP TABLE IF EXISTS palet');
        batch.execute('DROP TABLE IF EXISTS corepart');
        batch.execute('DROP TABLE IF EXISTS busbars');
        batch.execute('DROP TABLE IF EXISTS panels');
        batch.execute('DROP TABLE IF EXISTS company_accounts');
        batch.execute('DROP TABLE IF EXISTS companies');
        await batch.commit();
        await _onCreate(db, newVersion);
      },
    );
  }

  Future<void> _onCreate(Database db, int version) async {
    var batch = db.batch();

    batch.execute('''
      CREATE TABLE companies (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL UNIQUE,
        role TEXT NOT NULL
      )
    ''');
    batch.execute('''
      CREATE TABLE company_accounts (
        username TEXT PRIMARY KEY,
        password TEXT NOT NULL,
        company_id TEXT NOT NULL,
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE ON UPDATE CASCADE
      )
    ''');

    batch.execute('''
    CREATE TABLE panels (
      no_pp TEXT PRIMARY KEY, 
      no_panel TEXT NOT NULL UNIQUE,
      no_wbs TEXT NOT NULL,
      percent_progress REAL, 
      start_date TEXT, 
      target_delivery TEXT,
      status_busbar_pcc TEXT,
      status_busbar_mcc TEXT,
      status_component TEXT, 
      status_palet TEXT,  
      status_corepart TEXT,
      ao_busbar_pcc TEXT,
      ao_busbar_mcc TEXT,
      logs TEXT, 
      created_by TEXT NOT NULL,
      vendor_id TEXT, 
      is_closed INTEGER NOT NULL DEFAULT 0, 
      closed_date TEXT
    )
    ''');

    batch.execute('''
    CREATE TABLE busbars (
      id INTEGER PRIMARY KEY AUTOINCREMENT, 
      panel_no_pp TEXT NOT NULL,
      vendor TEXT NOT NULL, 
      remarks TEXT,
      UNIQUE(panel_no_pp, vendor)
    )
    ''');
    batch.execute('''
      CREATE TABLE components (
        id INTEGER PRIMARY KEY AUTOINCREMENT, 
        panel_no_pp TEXT NOT NULL,
        vendor TEXT NOT NULL,
        UNIQUE(panel_no_pp, vendor)
      )
    ''');
    batch.execute('''
      CREATE TABLE palet (
        id INTEGER PRIMARY KEY AUTOINCREMENT, 
        panel_no_pp TEXT NOT NULL,
        vendor TEXT NOT NULL,
        UNIQUE(panel_no_pp, vendor)
      )
    ''');
    batch.execute('''
      CREATE TABLE corepart (
        id INTEGER PRIMARY KEY AUTOINCREMENT, 
        panel_no_pp TEXT NOT NULL,
        vendor TEXT NOT NULL,
        UNIQUE(panel_no_pp, vendor)
      )
    ''');

    await _createDummyData(batch);
    await batch.commit(noResult: true);
  }

  Future<void> _createDummyData(Batch batch) async {
    final companies = [
      Company(id: 'admin', name: 'Administrator', role: AppRole.admin),
      Company(id: 'viewer', name: 'Viewer', role: AppRole.viewer),
      Company(id: 'warehouse', name: 'Warehouse', role: AppRole.warehouse),
      Company(id: 'abacus', name: 'Abacus', role: AppRole.k3),
      Company(id: 'gaa', name: 'GAA', role: AppRole.k3),
      Company(id: 'gpe', name: 'GPE', role: AppRole.k5),
      Company(id: 'dsm', name: 'DSM', role: AppRole.k5),
    ];
    for (final c in companies) {
      batch.insert('companies', c.toMap());
    }

    final accounts = [
      CompanyAccount(username: 'admin', password: '123', companyId: 'admin'),
      CompanyAccount(username: 'viewer', password: '123', companyId: 'viewer'),
      CompanyAccount(
        username: 'whs_user1',
        password: '123',
        companyId: 'warehouse',
      ),
      CompanyAccount(
        username: 'whs_user2',
        password: '123',
        companyId: 'warehouse',
      ),
      CompanyAccount(
        username: 'abacus_user1',
        password: '123',
        companyId: 'abacus',
      ),
      CompanyAccount(
        username: 'abacus_user2',
        password: '123',
        companyId: 'abacus',
      ),
      CompanyAccount(username: 'gaa_user1', password: '123', companyId: 'gaa'),
      CompanyAccount(username: 'gaa_user2', password: '123', companyId: 'gaa'),
      CompanyAccount(username: 'gpe_user1', password: '123', companyId: 'gpe'),
      CompanyAccount(username: 'gpe_user2', password: '123', companyId: 'gpe'),
      CompanyAccount(username: 'dsm_user1', password: '123', companyId: 'dsm'),
      CompanyAccount(username: 'dsm_user2', password: '123', companyId: 'dsm'),
    ];
    for (final a in accounts) {
      batch.insert('company_accounts', a.toMap());
    }

    final now = DateTime.now();
    final panels = [
      Panel(
        noPp: 'J-2101.01-A01-01',
        noPanel: 'MDP-01-GedungA',
        noWbs: 'WBS-A-001',
        percentProgress: 100,
        startDate: now.subtract(const Duration(days: 30)),
        targetDelivery: now.subtract(const Duration(days: 15)),
        statusBusbarPcc: 'Close',
        statusBusbarMcc: 'Close',
        statusComponent: 'Done',
        statusPalet: 'Close',
        statusCorepart: 'Close',
        createdBy: 'admin',
        vendorId: 'abacus',
        isClosed: true,
        closedDate: now.subtract(const Duration(days: 10)),
      ),
      Panel(
        noPp: 'J-2205.15-B02-02',
        noPanel: 'SDP-03-Lantai2',
        noWbs: 'WBS-B-002',
        percentProgress: 75,
        startDate: now.subtract(const Duration(days: 5)),
        targetDelivery: now.add(const Duration(days: 1)),
        statusBusbarPcc: 'On Progress',
        statusBusbarMcc: 'On Progress',
        statusComponent: 'On Progress',
        statusPalet: 'Close',
        statusCorepart: 'Close',
        createdBy: 'admin',
        vendorId: 'gaa',
        isClosed: false,
      ),
      Panel(
        noPp: 'J-2310.01-C11-03',
        noPanel: 'LVP-Gudang-Baru',
        noWbs: 'WBS-C-003',
        percentProgress: 10,
        startDate: now.subtract(const Duration(days: 1)),
        targetDelivery: now.add(const Duration(days: 10)),
        createdBy: 'admin',
        vendorId: 'abacus',
        isClosed: false,
      ),
    ];

    for (final p in panels) {
      batch.insert('panels', p.toMap());
      if (p.statusPalet == 'Close' && p.vendorId != null) {
        batch.insert('palet', {'panel_no_pp': p.noPp, 'vendor': p.vendorId});
      }
      if (p.statusCorepart == 'Close' && p.vendorId != null) {
        batch.insert('corepart', {'panel_no_pp': p.noPp, 'vendor': p.vendorId});
      }
    }

    batch.insert('busbars', {
      'panel_no_pp': 'J-2101.01-A01-01',
      'vendor': 'gpe',
      'remarks': 'Selesai',
    });
    batch.insert('components', {
      'panel_no_pp': 'J-2101.01-A01-01',
      'vendor': 'warehouse',
    });

    batch.insert('busbars', {
      'panel_no_pp': 'J-2205.15-B02-02',
      'vendor': 'dsm',
      'remarks': '',
    });
    batch.insert('components', {
      'panel_no_pp': 'J-2205.15-B02-02',
      'vendor': 'warehouse',
    });
  }

  Future<List<PanelDisplayData>> getAllPanelsForDisplay(
    Company currentUser,
  ) async {
    final db = await instance.database;

    String panelIdsSubQuery = '';
    List<dynamic> whereArgs = [];

    if (currentUser.role != AppRole.admin &&
        currentUser.role != AppRole.viewer) {
      switch (currentUser.role) {
        case AppRole.k3:
          panelIdsSubQuery = '''
            SELECT no_pp FROM panels WHERE vendor_id = ?
            UNION
            SELECT panel_no_pp FROM palet WHERE vendor = ?
            UNION
            SELECT panel_no_pp FROM corepart WHERE vendor = ?
          ''';
          whereArgs.addAll([currentUser.id, currentUser.id, currentUser.id]);
          break;
        case AppRole.k5:
          panelIdsSubQuery =
              'SELECT no_pp FROM panels WHERE no_pp IN (SELECT panel_no_pp FROM busbars WHERE vendor = ?)';
          whereArgs.add(currentUser.id);
          break;
        case AppRole.warehouse:
          panelIdsSubQuery =
              'SELECT no_pp FROM panels WHERE no_pp IN (SELECT panel_no_pp FROM components WHERE vendor = ?)';
          whereArgs.add(currentUser.id);
          break;
        default:
          return [];
      }
    } else {
      panelIdsSubQuery = 'SELECT no_pp FROM panels';
    }

    final String finalQuery =
        '''
      SELECT
        p.*,
        pu.name as panel_vendor_name,
        (SELECT GROUP_CONCAT(name) FROM companies WHERE id IN (SELECT vendor FROM busbars WHERE panel_no_pp = p.no_pp)) as busbar_vendor_names,
        (SELECT GROUP_CONCAT(id) FROM companies WHERE id IN (SELECT vendor FROM busbars WHERE panel_no_pp = p.no_pp)) as busbar_vendor_ids,
        (SELECT GROUP_CONCAT(remarks) FROM busbars WHERE panel_no_pp = p.no_pp) as busbar_remarks,
        (SELECT GROUP_CONCAT(name) FROM companies WHERE id IN (SELECT vendor FROM components WHERE panel_no_pp = p.no_pp)) as component_vendor_names,
        (SELECT GROUP_CONCAT(id) FROM companies WHERE id IN (SELECT vendor FROM components WHERE panel_no_pp = p.no_pp)) as component_vendor_ids,
        (SELECT GROUP_CONCAT(name) FROM companies WHERE id IN (SELECT vendor FROM palet WHERE panel_no_pp = p.no_pp)) as palet_vendor_names,
        (SELECT GROUP_CONCAT(id) FROM companies WHERE id IN (SELECT vendor FROM palet WHERE panel_no_pp = p.no_pp)) as palet_vendor_ids,
        (SELECT GROUP_CONCAT(name) FROM companies WHERE id IN (SELECT vendor FROM corepart WHERE panel_no_pp = p.no_pp)) as corepart_vendor_names,
        (SELECT GROUP_CONCAT(id) FROM companies WHERE id IN (SELECT vendor FROM corepart WHERE panel_no_pp = p.no_pp)) as corepart_vendor_ids
      FROM panels p
      LEFT JOIN companies pu ON p.vendor_id = pu.id
      WHERE p.no_pp IN ($panelIdsSubQuery)
      ORDER BY p.start_date DESC
    ''';

    final result = await db.rawQuery(finalQuery, whereArgs);
    return result.map((map) {
      List<String> cleanIds(String? rawIds) {
        if (rawIds == null || rawIds.isEmpty) return [];
        return rawIds.split(',').where((id) => id.isNotEmpty).toList();
      }

      return PanelDisplayData(
        panel: Panel.fromMap(map),
        panelVendorName: map['panel_vendor_name'] as String? ?? 'N/A',
        busbarVendorNames: map['busbar_vendor_names'] as String? ?? 'N/A',
        busbarVendorIds: cleanIds(map['busbar_vendor_ids'] as String?),
        busbarRemarks: map['busbar_remarks'] as String?,
        componentVendorNames: map['component_vendor_names'] as String? ?? 'N/A',
        componentVendorIds: cleanIds(map['component_vendor_ids'] as String?),
        paletVendorNames: map['palet_vendor_names'] as String? ?? 'N/A',
        paletVendorIds: cleanIds(map['palet_vendor_ids'] as String?),
        corepartVendorNames: map['corepart_vendor_names'] as String? ?? 'N/A',
        corepartVendorIds: cleanIds(map['corepart_vendor_ids'] as String?),
      );
    }).toList();
  }

  Future<Company?> login(String username, String password) async {
    final db = await instance.database;
    var res = await db.query(
      'company_accounts',
      where: 'username = ?',
      whereArgs: [username],
    );
    if (res.isNotEmpty) {
      var account = CompanyAccount.fromMap(res.first);
      if (account.password == password) {
        final companyRes = await db.query(
          'companies',
          where: 'id = ?',
          whereArgs: [account.companyId],
        );
        if (companyRes.isNotEmpty) {
          final company = Company.fromMap(companyRes.first);
          return company;
        }
      }
    }
    return null;
  }

  Future<Company?> getCompanyByUsername(String username) async {
    final db = await instance.database;
    final res = await db.rawQuery(
      '''
    SELECT c.*
    FROM companies c
    JOIN company_accounts ca ON c.id = ca.company_id
    WHERE ca.username = ?
  ''',
      [username],
    );

    if (res.isNotEmpty) {
      final company = Company.fromMap(res.first);
      return company;
    }
    return null;
  }

  Future<bool> updatePassword(String username, String newPassword) async {
    final db = await database;
    int count = await db.update(
      'company_accounts',
      {'password': newPassword},
      where: 'username = ?',
      whereArgs: [username],
    );
    return count > 0;
  }

  Future<void> insertCompanyWithAccount(
    Company company,
    CompanyAccount account,
  ) async {
    final db = await database;
    await db.transaction((txn) async {
      await txn.insert(
        'companies',
        company.toMap(),
        conflictAlgorithm: ConflictAlgorithm.ignore,
      );
      await txn.insert(
        'company_accounts',
        account.toMap(),
        conflictAlgorithm: ConflictAlgorithm.replace,
      );
    });
  }

  Future<void> updateCompanyAndAccount(
    Company company, {
    String? newPassword,
  }) async {
    final db = await database;
    await db.transaction((txn) async {
      final username = company.id;
      final newCompanyName = company.name;
      final newRole = company.role;

      final List<Map<String, dynamic>> companyRes = await txn.query(
        'companies',
        where: 'name = ?',
        whereArgs: [newCompanyName],
        limit: 1,
      );

      String targetCompanyId;
      if (companyRes.isEmpty) {
        targetCompanyId = newCompanyName.toLowerCase().replaceAll(
          RegExp(r'\s+'),
          '_',
        );
        await txn.insert('companies', {
          'id': targetCompanyId,
          'name': newCompanyName,
          'role': newRole.name,
        });
      } else {
        targetCompanyId = companyRes.first['id'] as String;
      }

      final Map<String, dynamic> accountUpdatePayload = {
        'company_id': targetCompanyId,
      };

      if (newPassword != null && newPassword.isNotEmpty) {
        accountUpdatePayload['password'] = newPassword;
      }

      await txn.update(
        'company_accounts',
        accountUpdatePayload,
        where: 'username = ?',
        whereArgs: [username],
      );
    });
  }

  Future<int> deleteCompanyAccount(String username) async {
    final db = await database;
    return await db.delete(
      'company_accounts',
      where: 'username = ?',
      whereArgs: [username],
    );
  }

  Future<Company?> getCompanyById(String id) async {
    final db = await instance.database;
    final res = await db.query('companies', where: 'id = ?', whereArgs: [id]);
    return res.isNotEmpty ? Company.fromMap(res.first) : null;
  }

  Future<List<Company>> getAllCompanies() async {
    final db = await database;
    final res = await db.query('companies');
    return res.map((e) => Company.fromMap(e)).toList();
  }

  Future<List<CompanyAccount>> getAllCompanyAccounts() async {
    final db = await database;
    final res = await db.query('company_accounts');
    return res.map((e) => CompanyAccount.fromMap(e)).toList();
  }

  Future<List<Map<String, dynamic>>> getAllUserAccountsForDisplay() async {
    final db = await database;
    final List<Map<String, dynamic>> maps = await db.rawQuery('''
      SELECT
          ca.username,
          ca.company_id,
          c.name AS company_name,
          c.role
      FROM company_accounts ca
      JOIN companies c ON ca.company_id = c.id
      WHERE ca.username != 'admin'
      ORDER BY c.name, ca.username
  ''');
    return maps;
  }

  Future<List<Map<String, dynamic>>> getColleagueAccountsForDisplay(
    String companyName,
    String currentUsername,
  ) async {
    final db = await database;
    final List<Map<String, dynamic>> maps = await db.rawQuery(
      '''
      SELECT
          ca.username,
          ca.company_id,
          c.name AS company_name,
          c.role
      FROM company_accounts ca
      JOIN companies c ON ca.company_id = c.id
      WHERE c.name = ? AND ca.username != ?
      ORDER BY ca.username
  ''',
      [companyName, currentUsername],
    );
    return maps;
  }

  Future<List<Map<String, String>>> getUniqueCompanyDataForForm() async {
    final db = await database;
    final List<Map<String, dynamic>> maps = await db.rawQuery('''
      SELECT name, role
      FROM companies
      WHERE role != 'admin'
      GROUP BY name
      ORDER BY name ASC
    ''');
    return maps.map((map) {
      return {'name': map['name'] as String, 'role': map['role'] as String};
    }).toList();
  }

  Future<Company?> getCompanyByName(String name) async {
    final db = await database;
    final res = await db.query(
      'companies',
      where: 'name = ?',
      whereArgs: [name],
      limit: 1,
    );
    return res.isNotEmpty ? Company.fromMap(res.first) : null;
  }

  Future<List<Company>> getK3Vendors() => _getCompaniesByRole(AppRole.k3);
  Future<List<Company>> getK5Vendors() => _getCompaniesByRole(AppRole.k5);
  Future<List<Company>> getWHSVendors() =>
      _getCompaniesByRole(AppRole.warehouse);

  Future<List<Company>> _getCompaniesByRole(AppRole role) async {
    final db = await instance.database;
    final result = await db.query(
      'companies',
      where: 'role = ?',
      whereArgs: [role.name],
    );
    return result.map((map) => Company.fromMap(map)).toList();
  }

  Future<int> insertPanel(Panel panel) async {
    Database db = await instance.database;
    return await db.insert('panels', panel.toMap());
  }

  Future<List<Panel>> getAllPanels() async {
    Database db = await instance.database;
    return (await db.query('panels')).map((map) => Panel.fromMap(map)).toList();
  }

  Future<Panel?> getPanelByNoPp(String noPp) async {
    Database db = await instance.database;
    var result = await db.query(
      'panels',
      where: 'no_pp = ?',
      whereArgs: [noPp],
    );
    return result.isNotEmpty ? Panel.fromMap(result.first) : null;
  }

  Future<int> updatePanel(Panel panel) async {
    Database db = await instance.database;
    return await db.update(
      'panels',
      panel.toMap(),
      where: 'no_pp = ?',
      whereArgs: [panel.noPp],
    );
  }

  Future<int> deletePanel(String noPp) async {
    Database db = await instance.database;
    return await db.delete('panels', where: 'no_pp = ?', whereArgs: [noPp]);
  }

  Future<int> upsertBusbar(Busbar busbar) async {
    final db = await database;
    return await db.insert(
      'busbars',
      busbar.toMap(),
      conflictAlgorithm: ConflictAlgorithm.ignore,
    );
  }

  Future<int> upsertComponent(Component component) async {
    final db = await database;
    return await db.insert(
      'components',
      component.toMap(),
      conflictAlgorithm: ConflictAlgorithm.ignore,
    );
  }

  Future<int> upsertPalet(Palet palet) async {
    final db = await database;
    return await db.insert(
      'palet',
      palet.toMap(),
      conflictAlgorithm: ConflictAlgorithm.ignore,
    );
  }

  Future<int> upsertCorepart(Corepart corepart) async {
    final db = await database;
    return await db.insert(
      'corepart',
      corepart.toMap(),
      conflictAlgorithm: ConflictAlgorithm.ignore,
    );
  }

  Future<List<Component>> getAllComponents() async {
    Database db = await instance.database;
    return (await db.query(
      'components',
    )).map((map) => Component.fromMap(map)).toList();
  }

  Future<List<Palet>> getAllPalet() async {
    Database db = await instance.database;
    return (await db.query('palet')).map((map) => Palet.fromMap(map)).toList();
  }

  Future<List<Corepart>> getAllCorepart() async {
    Database db = await instance.database;
    return (await db.query(
      'corepart',
    )).map((map) => Corepart.fromMap(map)).toList();
  }

  Future<void> upsertBusbarRemarkandVendor({
    required String panelNoPp,
    required String vendorId,
    required String newRemark,
  }) async {
    final db = await database;
    final existing = await db.query(
      'busbars',
      where: 'panel_no_pp = ? AND vendor = ?',
      whereArgs: [panelNoPp, vendorId],
    );
    if (existing.isEmpty) {
      await db.insert('busbars', {
        'panel_no_pp': panelNoPp,
        'vendor': vendorId,
        'remarks': newRemark,
      });
    } else {
      await db.update(
        'busbars',
        {'remarks': newRemark},
        where: 'panel_no_pp = ? AND vendor = ?',
        whereArgs: [panelNoPp, vendorId],
      );
    }
  }

  Future<List<Busbar>> getAllBusbars() async {
    Database db = await instance.database;
    return (await db.query(
      'busbars',
    )).map((map) => Busbar.fromMap(map)).toList();
  }

  Future<Map<String, List<dynamic>>> getFilteredDataForExport(
    Company currentUser,
  ) async {
    final db = await database;

    List<Company> companies = [];
    List<CompanyAccount> companyAccounts = [];
    List<Panel> panels = [];
    List<Busbar> busbars = [];
    List<Component> components = [];
    List<Palet> palet = [];
    List<Corepart> corepart = [];

    if (currentUser.role == AppRole.admin ||
        currentUser.role == AppRole.viewer) {
      companies = await getAllCompanies();
      companyAccounts = await getAllCompanyAccounts();
      panels = await getAllPanels();
      busbars = await getAllBusbars();
      components = await getAllComponents();
      palet = await getAllPalet();
      corepart = await getAllCorepart();
    } else {
      List<String> relevantPanelIds = [];
      String companyId = currentUser.id;

      final currentUserCompany = await getCompanyById(currentUser.id);
      if (currentUserCompany != null) {
        companies.add(currentUserCompany);
      }

      final currentUserAccounts = await db.query(
        'company_accounts',
        where: 'company_id = ?',
        whereArgs: [currentUser.id],
      );
      companyAccounts.addAll(
        currentUserAccounts.map(CompanyAccount.fromMap).toList(),
      );

      switch (currentUser.role) {
        case AppRole.k3:
          final resPanels = await db.query(
            'panels',
            columns: ['no_pp'],
            where: 'vendor_id = ?',
            whereArgs: [companyId],
          );
          relevantPanelIds.addAll(
            resPanels.map((map) => map['no_pp'] as String).toList(),
          );

          final resPalet = await db.query(
            'palet',
            distinct: true,
            columns: ['panel_no_pp'],
            where: 'vendor = ?',
            whereArgs: [companyId],
          );
          relevantPanelIds.addAll(
            resPalet.map((map) => map['panel_no_pp'] as String).toList(),
          );

          final resCorepart = await db.query(
            'corepart',
            distinct: true,
            columns: ['panel_no_pp'],
            where: 'vendor = ?',
            whereArgs: [companyId],
          );
          relevantPanelIds.addAll(
            resCorepart.map((map) => map['panel_no_pp'] as String).toList(),
          );
          break;

        case AppRole.k5:
          final resBusbars = await db.query(
            'busbars',
            distinct: true,
            columns: ['panel_no_pp'],
            where: 'vendor = ?',
            whereArgs: [companyId],
          );
          relevantPanelIds = resBusbars
              .map((map) => map['panel_no_pp'] as String)
              .toList();
          break;

        case AppRole.warehouse:
          final resComponents = await db.query(
            'components',
            distinct: true,
            columns: ['panel_no_pp'],
            where: 'vendor = ?',
            whereArgs: [companyId],
          );
          relevantPanelIds = resComponents
              .map((map) => map['panel_no_pp'] as String)
              .toList();
          break;
        default:
          break;
      }

      relevantPanelIds = relevantPanelIds.toSet().toList();

      if (relevantPanelIds.isNotEmpty) {
        String placeholders = List.filled(
          relevantPanelIds.length,
          '?',
        ).join(',');

        panels = (await db.query(
          'panels',
          where: 'no_pp IN ($placeholders)',
          whereArgs: relevantPanelIds,
        )).map(Panel.fromMap).toList();

        busbars = (await db.query(
          'busbars',
          where: 'panel_no_pp IN ($placeholders)',
          whereArgs: relevantPanelIds,
        )).map(Busbar.fromMap).toList();

        components = (await db.query(
          'components',
          where: 'panel_no_pp IN ($placeholders)',
          whereArgs: relevantPanelIds,
        )).map(Component.fromMap).toList();
        palet = (await db.query(
          'palet',
          where: 'panel_no_pp IN ($placeholders)',
          whereArgs: relevantPanelIds,
        )).map(Palet.fromMap).toList();
        corepart = (await db.query(
          'corepart',
          where: 'panel_no_pp IN ($placeholders)',
          whereArgs: relevantPanelIds,
        )).map(Corepart.fromMap).toList();
      }
    }

    return {
      'companies': companies,
      'companyAccounts': companyAccounts,
      'panels': panels,
      'busbars': busbars,
      'components': components,
      'palet': palet,
      'corepart': corepart,
    };
  }

  Future<Excel> generateFilteredDatabaseExcel(
    Map<String, bool> tablesToInclude,
    Company currentUser,
  ) async {
    final excel = Excel.createExcel();
    excel.delete('Sheet1');

    final data = await getFilteredDataForExport(currentUser);

    CellValue? _toCellValue(dynamic value) {
      if (value == null) return null;
      if (value is String) return TextCellValue(value);
      if (value is int) return IntCellValue(value);
      if (value is double) return DoubleCellValue(value);
      if (value is bool) return BoolCellValue(value);
      return TextCellValue(value.toString());
    }

    if (tablesToInclude['Companies'] == true) {
      final sheet = excel['Companies'];
      sheet.appendRow([
        TextCellValue('id'),
        TextCellValue('name'),
        TextCellValue('role'),
      ]);
      for (final item in data['companies'] as List<Company>) {
        sheet.appendRow([
          _toCellValue(item.id),
          _toCellValue(item.name),
          _toCellValue(item.role.name),
        ]);
      }
    }

    if (tablesToInclude['Company Accounts'] == true) {
      final sheet = excel['Company Accounts'];
      sheet.appendRow([
        TextCellValue('username'),
        TextCellValue('password'),
        TextCellValue('company_id'),
      ]);
      for (final item in data['companyAccounts'] as List<CompanyAccount>) {
        sheet.appendRow([
          _toCellValue(item.username),
          _toCellValue(item.password),
          _toCellValue(item.companyId),
        ]);
      }
    }

    if (tablesToInclude['Panels'] == true) {
      final sheet = excel['Panels'];
      sheet.appendRow([
        TextCellValue('no_pp'),
        TextCellValue('no_panel'),
        TextCellValue('no_wbs'),
        TextCellValue('percent_progress'),
        TextCellValue('start_date'),
        TextCellValue('target_delivery'),
        TextCellValue('status_busbar_pcc'),
        TextCellValue('status_busbar_mcc'),
        TextCellValue('status_component'),
        TextCellValue('status_palet'),
        TextCellValue('status_corepart'),
        TextCellValue('ao_busbar_pcc'),
        // TextCellValue('eta_busbar_pcc'),
        TextCellValue('ao_busbar_mcc'),
        // TextCellValue('eta_busbar_mcc'),
        // TextCellValue('ao_component'),
        // TextCellValue('eta_component'),
        TextCellValue('created_by'),
        TextCellValue('vendor_id'),
        TextCellValue('is_closed'),
        TextCellValue('closed_date'),
      ]);
      for (final p in data['panels'] as List<Panel>) {
        sheet.appendRow([
          _toCellValue(p.noPp),
          _toCellValue(p.noPanel),
          _toCellValue(p.noWbs),
          _toCellValue(p.percentProgress),
          _toCellValue(p.startDate?.toIso8601String()),
          _toCellValue(p.targetDelivery?.toIso8601String()),
          _toCellValue(p.statusBusbarPcc),
          _toCellValue(p.statusBusbarMcc),
          _toCellValue(p.statusComponent),
          _toCellValue(p.statusPalet),
          _toCellValue(p.statusCorepart),
          _toCellValue(p.aoBusbarPcc?.toIso8601String()),
          // _toCellValue(p.etaBusbarPcc?.toIso8601String()),
          _toCellValue(p.aoBusbarMcc?.toIso8601String()),
          // _toCellValue(p.etaBusbarMcc?.toIso8601String()),
          // _toCellValue(p.aoComponent?.toIso8601String()),
          // _toCellValue(p.etaComponent?.toIso8601String()),
          _toCellValue(p.createdBy),
          _toCellValue(p.vendorId),
          _toCellValue(p.isClosed),
          _toCellValue(p.closedDate?.toIso8601String()),
        ]);
      }
    }

    if (tablesToInclude['Busbars'] == true) {
      final sheet = excel['Busbars'];
      sheet.appendRow([
        TextCellValue('panel_no_pp'),
        TextCellValue('vendor'),
        TextCellValue('remarks'),
      ]);
      for (final b in data['busbars'] as List<Busbar>) {
        sheet.appendRow([
          _toCellValue(b.panelNoPp),
          _toCellValue(b.vendor),
          _toCellValue(b.remarks),
        ]);
      }
    }

    if (tablesToInclude['Components'] == true) {
      final sheet = excel['Components'];
      sheet.appendRow([TextCellValue('panel_no_pp'), TextCellValue('vendor')]);
      for (final c in data['components'] as List<Component>) {
        sheet.appendRow([_toCellValue(c.panelNoPp), _toCellValue(c.vendor)]);
      }
    }
    if (tablesToInclude['Palet'] == true) {
      final sheet = excel['Palet'];
      sheet.appendRow([TextCellValue('panel_no_pp'), TextCellValue('vendor')]);
      for (final c in data['palet'] as List<Palet>) {
        sheet.appendRow([_toCellValue(c.panelNoPp), _toCellValue(c.vendor)]);
      }
    }
    if (tablesToInclude['Corepart'] == true) {
      final sheet = excel['Corepart'];
      sheet.appendRow([TextCellValue('panel_no_pp'), TextCellValue('vendor')]);
      for (final c in data['corepart'] as List<Corepart>) {
        sheet.appendRow([_toCellValue(c.panelNoPp), _toCellValue(c.vendor)]);
      }
    }
    return excel;
  }

  Future<String> generateFilteredDatabaseJson(
    Map<String, bool> tablesToInclude,
    Company currentUser,
  ) async {
    final Map<String, dynamic> jsonData = {};
    final data = await getFilteredDataForExport(currentUser);

    if (tablesToInclude['Companies'] == true) {
      jsonData['companies'] = (data['companies'] as List<Company>)
          .map((e) => e.toMap())
          .toList();
    }
    if (tablesToInclude['Company Accounts'] == true) {
      jsonData['company_accounts'] =
          (data['companyAccounts'] as List<CompanyAccount>)
              .map((e) => e.toMap())
              .toList();
    }
    if (tablesToInclude['Panels'] == true) {
      jsonData['panels'] = (data['panels'] as List<Panel>)
          .map((p) => p.toMap())
          .toList();
    }
    if (tablesToInclude['Busbars'] == true) {
      jsonData['busbars'] = (data['busbars'] as List<Busbar>)
          .map((b) => b.toMap())
          .toList();
    }
    if (tablesToInclude['Components'] == true) {
      jsonData['components'] = (data['components'] as List<Component>)
          .map((c) => c.toMap())
          .toList();
    }
    if (tablesToInclude['Palet'] == true) {
      jsonData['palet'] = (data['palet'] as List<Palet>)
          .map((c) => c.toMap())
          .toList();
    }
    if (tablesToInclude['Corepart'] == true) {
      jsonData['corepart'] = (data['corepart'] as List<Corepart>)
          .map((c) => c.toMap())
          .toList();
    }
    return JsonEncoder.withIndent('  ').convert(jsonData);
  }

  Future<void> importData(
    Map<String, List<Map<String, dynamic>>> data,
    Function(double progress, String message) onProgress,
  ) async {
    final db = await database;
    await db.transaction((txn) async {
      int totalOperations = data.values.fold(
        0,
        (sum, list) => sum + list.length,
      );
      if (totalOperations == 0) return;
      int completedOperations = 0;

      void updateProgress(String message) {
        completedOperations++;
        onProgress(completedOperations / totalOperations, message);
      }

      if (data.containsKey('companies') && data['companies'] != null) {
        for (var itemData in data['companies']!) {
          updateProgress(
            "Importing company: ${itemData['name'] ?? itemData['id']}",
          );
          await txn.insert(
            'companies',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
      if (data.containsKey('company_accounts') &&
          data['company_accounts'] != null) {
        for (var itemData in data['company_accounts']!) {
          updateProgress("Importing account: ${itemData['username']}");
          await txn.insert(
            'company_accounts',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
      if (data.containsKey('panels') && data['panels'] != null) {
        for (var itemData in data['panels']!) {
          updateProgress("Importing panel: ${itemData['no_panel']}");
          await txn.insert(
            'panels',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
      if (data.containsKey('busbars') && data['busbars'] != null) {
        for (var itemData in data['busbars']!) {
          updateProgress("Linking busbar: ${itemData['panel_no_pp']}");
          await txn.insert(
            'busbars',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
      if (data.containsKey('components') && data['components'] != null) {
        for (var itemData in data['components']!) {
          updateProgress("Linking component: ${itemData['panel_no_pp']}");
          await txn.insert(
            'components',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
      if (data.containsKey('palet') && data['palet'] != null) {
        for (var itemData in data['palet']!) {
          updateProgress("Linking palet: ${itemData['panel_no_pp']}");
          await txn.insert(
            'palet',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
      if (data.containsKey('corepart') && data['corepart'] != null) {
        for (var itemData in data['corepart']!) {
          updateProgress("Linking corepart: ${itemData['panel_no_pp']}");
          await txn.insert(
            'corepart',
            itemData,
            conflictAlgorithm: ConflictAlgorithm.replace,
          );
        }
      }
    });
  }

  Future<TemplateFile> generateImportTemplate({
    required String dataType,
    required String format,
  }) async {
    if (format == 'json') {
      final jsonString = _generateJsonTemplateString(dataType);
      return TemplateFile(bytes: utf8.encode(jsonString), extension: 'json');
    } else {
      final excel = _generateExcelTemplate(dataType);
      final bytes = excel.encode();
      if (bytes == null) throw Exception("Gagal membuat file Excel.");
      return TemplateFile(bytes: bytes, extension: 'xlsx');
    }
  }

  String _generateJsonTemplateString(String dataType) {
    Map<String, dynamic> jsonData = {};

    if (dataType == 'companies_and_accounts') {
      jsonData['companies'] = [
        {
          'id': 'vendor_k3_contoh',
          'name': 'Nama Vendor K3 Contoh',
          'role': "pilih: k3, k5, warehouse, viewer",
        },
      ];
      jsonData['company_accounts'] = [
        {
          'username': 'staff_k3_contoh',
          'password': '123',
          'company_id': 'vendor_k3_contoh',
        },
      ];
    } else {
      final now = DateTime.now().toIso8601String();
      jsonData['panels'] = [
        {
          'no_pp': 'PP-CONTOH-01',
          'no_panel': 'PANEL-CONTOH-A',
          'no_wbs': 'WBS-CONTOH-01',
          'percent_progress': 80.5,
          'start_date': now,
          'target_delivery': now,
          'status_busbar_pcc': 'On Progress',
          'status_busbar_mcc': 'Open',
          'status_component': 'Open',
          'status_palet': 'Open',
          'status_corepart': 'Open',
          'ao_busbar_pcc': now,
          // 'eta_busbar_pcc': now,
          'ao_busbar_mcc': now,
          // 'eta_busbar_mcc': now,
          // 'ao_component': now,
          // 'eta_component': now,
          'created_by': 'admin',
          'vendor_id': 'vendor_k3_contoh',
          'is_closed': 0,
          'closed_date': null,
        },
      ];
      jsonData['busbars'] = [
        {
          'panel_no_pp': 'PP-CONTOH-01',
          'vendor': 'vendor_k5_contoh',
          'remarks': 'Catatan untuk busbar',
        },
      ];
      jsonData['components'] = [
        {'panel_no_pp': 'PP-CONTOH-01', 'vendor': 'warehouse'},
      ];
      jsonData['palet'] = [
        {'panel_no_pp': 'PP-CONTOH-01', 'vendor': 'vendor_k3_contoh'},
      ];
      jsonData['corepart'] = [
        {'panel_no_pp': 'PP-CONTOH-01', 'vendor': 'vendor_k3_contoh'},
      ];
    }
    return JsonEncoder.withIndent('  ').convert(jsonData);
  }

  Excel _generateExcelTemplate(String dataType) {
    final excel = Excel.createExcel();
    excel.delete('Sheet1');

    CellValue? _toCellValue(dynamic value) {
      if (value == null) return null;
      if (value is String) return TextCellValue(value);
      if (value is int) return IntCellValue(value);
      if (value is double) return DoubleCellValue(value);
      if (value is bool) return BoolCellValue(value);
      return TextCellValue(value.toString());
    }

    if (dataType == 'companies_and_accounts') {
      final companySheet = excel['companies'];
      companySheet.appendRow([
        TextCellValue('id'),
        TextCellValue('name'),
        TextCellValue('role'),
      ]);
      companySheet.appendRow([
        _toCellValue('vendor_k3_contoh'),
        _toCellValue('Nama Vendor K3 Contoh'),
        _toCellValue('k3'),
      ]);

      final accountSheet = excel['company_accounts'];
      accountSheet.appendRow([
        TextCellValue('username'),
        TextCellValue('password'),
        TextCellValue('company_id'),
      ]);
      accountSheet.appendRow([
        _toCellValue('staff_k3_contoh'),
        _toCellValue('password123'),
        _toCellValue('vendor_k3_contoh'),
      ]);
    } else {
      final panelSheet = excel['panels'];
      panelSheet.appendRow([
        TextCellValue('no_pp'),
        TextCellValue('no_panel'),
        TextCellValue('no_wbs'),
        TextCellValue('percent_progress'),
        TextCellValue('start_date'),
        TextCellValue('target_delivery'),
        TextCellValue('status_busbar_pcc'),
        TextCellValue('status_busbar_mcc'),
        TextCellValue('status_component'),
        TextCellValue('status_palet'),
        TextCellValue('status_corepart'),
        TextCellValue('ao_busbar_pcc'),
        // TextCellValue('eta_busbar_pcc'),
        TextCellValue('ao_busbar_mcc'),
        // TextCellValue('eta_busbar_mcc'),
        // TextCellValue('ao_component'),
        // TextCellValue('eta_component'),
        TextCellValue('created_by'),
        TextCellValue('vendor_id'),
        TextCellValue('is_closed'),
        TextCellValue('closed_date'),
      ]);

      final busbarSheet = excel['busbars'];
      busbarSheet.appendRow([
        TextCellValue('panel_no_pp'),
        TextCellValue('vendor'),
        TextCellValue('remarks'),
      ]);

      final componentSheet = excel['components'];
      componentSheet.appendRow([
        TextCellValue('panel_no_pp'),
        TextCellValue('vendor'),
      ]);

      final paletSheet = excel['palet'];
      paletSheet.appendRow([
        TextCellValue('panel_no_pp'),
        TextCellValue('vendor'),
      ]);

      final corepartSheet = excel['corepart'];
      corepartSheet.appendRow([
        TextCellValue('panel_no_pp'),
        TextCellValue('vendor'),
      ]);
    }
    return excel;
  }

  Future<bool> isPanelNumberUnique(
    String noPanel, {
    String? currentNoPp,
  }) async {
    final db = await database;
    List<Map<String, dynamic>> result;
    if (currentNoPp != null) {
      result = await db.query(
        'panels',
        where: 'no_panel = ? AND no_pp != ?',
        whereArgs: [noPanel, currentNoPp],
        limit: 1,
      );
    } else {
      result = await db.query(
        'panels',
        where: 'no_panel = ?',
        whereArgs: [noPanel],
        limit: 1,
      );
    }
    return result.isEmpty;
  }

  Future<bool> isUsernameTaken(String username) async {
    final db = await database;
    final result = await db.query(
      'company_accounts',
      where: 'username = ?',
      whereArgs: [username],
      limit: 1,
    );
    return result.isNotEmpty;
  }
}
