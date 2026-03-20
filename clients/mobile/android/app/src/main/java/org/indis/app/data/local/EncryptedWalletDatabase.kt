package org.indis.app.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Entity
import androidx.room.PrimaryKey
import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import androidx.room.Room
import androidx.room.RoomDatabase

/**
 * Room entity representing a single W3C Verifiable Credential stored locally.
 *
 * Raw VC JSON is stored in [vcJson].  In a production build the database file
 * is encrypted via SQLCipher using a key derived from the user's biometric +
 * hardware-backed Android Keystore key (deferred until production hardening).
 *
 * PRD FR-006: credentials must be available offline for up to 72 hours.
 */
@Entity(tableName = "credentials")
data class CredentialEntity(
    @PrimaryKey val id: String,
    val credentialType: String,
    val issuer: String,
    val issuedAt: String,
    val expiresAt: String,
    /** Full W3C VC JSON string. */
    val vcJson: String,
    /** Base64-encoded ZK proof bytes (optional). */
    val proofB64: String? = null,
    val status: String = "active",
)

@Dao
interface CredentialDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(credential: CredentialEntity)

    @Query("SELECT * FROM credentials ORDER BY issuedAt DESC")
    suspend fun getAllCredentials(): List<CredentialEntity>

    @Query("SELECT * FROM credentials WHERE id = :id LIMIT 1")
    suspend fun getById(id: String): CredentialEntity?

    @Query("DELETE FROM credentials WHERE id = :id")
    suspend fun deleteById(id: String)

    @Query("DELETE FROM credentials")
    suspend fun clearAll()
}

@Database(entities = [CredentialEntity::class], version = 1, exportSchema = false)
abstract class EncryptedWalletDatabase : RoomDatabase() {

    abstract fun credentialDao(): CredentialDao

    companion object {
        @Volatile private var INSTANCE: EncryptedWalletDatabase? = null

        fun getInstance(context: Context): EncryptedWalletDatabase {
            return INSTANCE ?: synchronized(this) {
                // TODO(production): replace with SQLCipher-backed SupportFactory.
                // For local dev, plain Room is sufficient.
                Room.databaseBuilder(
                    context.applicationContext,
                    EncryptedWalletDatabase::class.java,
                    "indis_wallet.db",
                ).fallbackToDestructiveMigration().build().also { INSTANCE = it }
            }
        }
    }
}
