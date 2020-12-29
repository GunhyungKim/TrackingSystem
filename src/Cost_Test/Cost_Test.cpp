/* Copyright (C) 2019 IBM Corp.
 * This program is Licensed under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License. See accompanying LICENSE file.
 */

// This is a sample program for education purposes only.
// It attempts to demonstrate the use of the API for the
// binary arithmetic operations that can be performed.

#include <iostream>
#include <chrono>

#include <helib/helib.h>
#include <helib/binaryArith.h>
#include <helib/intraSlot.h>

int main(int argc, char* argv[])
{
  /*  Example of binary arithmetic using the BGV scheme  */

  // First set up parameters.
  //
  // NOTE: The parameters used in this example code are for demonstration only.
  // They were chosen to provide the best performance of execution while
  // providing the context to demonstrate how to use the "Binary Arithmetic
  // APIs". The parameters do not provide the security level that might be
  // required by real use/application scenarios.

  // Plaintext prime modulus.
  long p = 2;
  // Cyclotomic polynomial - defines phi(m).
  long m = 4095;
  // Hensel lifting (default = 1).
  long r = 1;
  // Number of bits of the modulus chain.
  long bits = 500;
  // Number of columns of Key-Switching matrix (typically 2 or 3).
  long c = 2;
  // Factorisation of m required for bootstrapping.
  std::vector<long> mvec = {7, 5, 9, 13};
  // Generating set of Zm* group.
  std::vector<long> gens = {2341, 3277, 911};
  // Orders of the previous generators.
  std::vector<long> ords = {6, 4, 6};

  std::cout << "Initialising context object..." << std::endl;
  // Initialize the context.
  // This object will hold information about the algebra created from the
  // previously set parameters.
  helib::Context context(m, p, r, gens, ords);

  // Modify the context, adding primes to the modulus chain.
  // This defines the ciphertext space.
  std::cout << "Building modulus chain..." << std::endl;
  buildModChain(context, bits, c);

  // Print the context.
  context.zMStar.printout();
  std::cout << std::endl;

  // Print the security level.
  std::cout << "Security: " << context.securityLevel() << std::endl;

  // Secret key management.
  std::cout << "Creating secret key..." << std::endl;
  // Create a secret key associated with the context.
  helib::SecKey secret_key(context);
  // Generate the secret key.
  secret_key.GenSecKey();

  // Public key management.
  // Set the secret key (upcast: SecKey is a subclass of PubKey).
  const helib::PubKey& public_key = secret_key;

  // Get the EncryptedArray of the context.
  const helib::EncryptedArray& ea = *(context.ea);

  // Build the unpack slot encoding.
  std::vector<helib::zzX> unpackSlotEncoding;
  buildUnpackSlotEncoding(unpackSlotEncoding, ea);

  // Get the number of slot (phi(m)).
  long nslots = ea.size();
  std::cout << "Number of slots: " << nslots << std::endl;

  // Generate three random binary numbers a, b, c.
  // Encrypt them under BGV.
  // Calculate a * b + c with HElib's binary arithmetic functions, then decrypt
  // the result.
  // Next, calculate a + b + c with HElib's binary arithmetic functions, then
  // decrypt the result.
  // Finally, calculate popcnt(a) with HElib's binary arithmetic functions,
  // then decrypt the result.  Note that popcnt, also known as hamming weight
  // or bit summation, returns the count of non-zero bits.

  // Each bit of the binary number is encoded into a single ciphertext. Thus
  // for a 16 bit binary number, we will represent this as an array of 16
  // unique ciphertexts.
  // i.e. b0 = [0] [0] [0] ... [0] [0] [0]        ciphertext for bit 0
  //      b1 = [1] [1] [1] ... [1] [1] [1]        ciphertext for bit 1
  //      b2 = [1] [1] [1] ... [1] [1] [1]        ciphertext for bit 2
  // These 3 ciphertexts represent the 3-bit binary number 110b = 6

  // Note: several numbers can be encoded across the slots of each ciphertext
  // which would result in several parallel slot-wise operations.
  // For simplicity we place the same data into each slot of each ciphertext,
  // printing out only the back of each vector.
  // NB: fifteenOrLess4Four max is 15 bits. Later in the code we pop the MSB.
  long bitSize = 63;
  long strSize = 1;
  long outSize = 2 * bitSize;
  long a_data = NTL::RandomBits_long(bitSize);
  std::string s;
  s="010-1234-5678";
  s.resize(strSize);

  std::cout << "Pre-encryption data" << std::endl;
  std::cout << "s = " << s << std::endl;
  std::cout << "a = " << a_data << std::endl;

  // Use a scratch ciphertext to populate vectors.
  helib::Ctxt scratch(public_key);
  std::vector<helib::Ctxt> encrypted_a(bitSize, scratch);

  std::vector<helib::Ctxt> encrypted_s(bitSize, scratch);
  std::vector<std::vector<helib::Ctxt>> enc_char(strSize, encrypted_s);

  // Encrypt the data in binary representation.
  std::chrono::high_resolution_clock::time_point start1 = std::chrono::high_resolution_clock::now();
 for (long i = 0; i < bitSize; ++i) {
    std::vector<long> a_vec(ea.size());
    // Extract the i'th bit of a,b,c.
    for (auto& slot : a_vec)
      slot = (a_data >> i) & 1;
    ea.encrypt(encrypted_a[i], public_key, a_vec);
  }
  std::chrono::high_resolution_clock::time_point end1 = std::chrono::high_resolution_clock::now();

  std::chrono::duration<double> diff_enc1 = std::chrono::duration_cast<std::chrono::duration<double>>(end1-start1);

  std::chrono::high_resolution_clock::time_point start = std::chrono::high_resolution_clock::now();

  for (long i=0; i<strSize; i++) {
    for (long j=0; j<bitSize; j++) {
      std::vector<long> s_vec(ea.size());
      for(auto& slot : s_vec)
        slot = (s.at(i) >> j) & 1;
      ea.encrypt(enc_char[i][j], public_key, s_vec);
    }
  }

  std::chrono::high_resolution_clock::time_point end = std::chrono::high_resolution_clock::now();
  std::chrono::duration<double> diff_enc = std::chrono::duration_cast<std::chrono::duration<double>>(end-start);

  // Although in general binary numbers are represented here as
  // std::vector<helib::Ctxt> the binaryArith APIs for HElib use the PtrVector
  // wrappers instead, e.g. helib::CtPtrs_vectorCt. These are nothing more than
  // thin wrapper classes to standardise access to different vector types, such
  // as NTL::Vec and std::vector. They do not take ownership of the underlying
  // object but merely provide access to it.
  //
  // helib::CtPtrMat_vectorCt is a wrapper for
  // std::vector<std::vector<helib::Ctxt>>, used for representing a list of
  // encrypted binary numbers.

  // Perform the multiplication first and put it in encrypted_product.
  std::vector<helib::Ctxt> encrypted_c(bitSize, scratch);
  std::vector<std::vector<helib::Ctxt>> enc_mid(strSize, encrypted_c);
  std::vector<std::vector<helib::Ctxt>> enc_end(strSize, encrypted_c);

  start = std::chrono::high_resolution_clock::now();

  for(long i=0; i<strSize; i++) {
    helib::CtPtrs_vectorCt middle_wrapper(enc_mid[i]);

    helib::bitwiseXOR(
      middle_wrapper,
      helib::CtPtrs_vectorCt(enc_char[i]),
      helib::CtPtrs_vectorCt(encrypted_a)
    );
  }

  end = std::chrono::high_resolution_clock::now();
  std::chrono::duration<double> diff_bit = std::chrono::duration_cast<std::chrono::duration<double>>(end-start);


  std::cout << "Operation XOR" << std::endl;
  // Decrypt and print the result.
  for (long i=0; i<strSize; i++){
    helib::CtPtrs_vectorCt middle_wrapper(enc_mid[i]);
    std::vector<long> decrypted_middle;
    helib::decryptBinaryNums(decrypted_middle, middle_wrapper, secret_key, ea);
    std::cout << (char)decrypted_middle.back();
  }
  std::cout << std::endl;

  for(long i=0; i<strSize; i++) {
    helib::CtPtrs_vectorCt result_wrapper(enc_end[i]);

    helib::bitwiseXOR(
      result_wrapper,
      helib::CtPtrs_vectorCt(enc_mid[i]),
      helib::CtPtrs_vectorCt(encrypted_a)
    );
  }

  std::cout << "Operation XOR XOR" << std::endl;
  // Decrypt and print the result.
  start = std::chrono::high_resolution_clock::now();
  for(long i=0; i<strSize; i++) {
    helib::CtPtrs_vectorCt result_wrapper(enc_end[i]);

    std::vector<long> decrypted_result;
    helib::decryptBinaryNums(decrypted_result, result_wrapper, secret_key, ea);
    std::cout << (char)decrypted_result.back();
  }
  std::cout << std::endl;
  end = std::chrono::high_resolution_clock::now();
  std::chrono::duration<double> diff_dec = std::chrono::duration_cast<std::chrono::duration<double>>(end-start);

  std::cout << "encryption 1byte char : " << diff_enc1.count() << "seconds" << std::endl;
  std::cout << "encryption 6byte char : " << diff_enc.count() << "seconds" << std::endl;
  std::cout << "bitwise op 6byte char : " << diff_bit.count() << "seconds" << std::endl;
  std::cout << "decryption 6byte char : " << diff_dec.count() << "seconds" << std::endl;


  return 0;
}
